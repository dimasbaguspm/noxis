// Package docker is the docker CLI wrapper: every docker command the app
// issues lives here so feature code never builds shell invocations. Runs via
// the docker socket (CLI binary inside containers).
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"sort"
	"strings"
)

// Runner executes docker commands; swapped for a fake in tests.
type Runner func(ctx context.Context, args ...string) ([]byte, error)

type Client struct{ run Runner }

// New returns a client running `docker` on the local socket.
func New() *Client {
	return &Client{run: func(ctx context.Context, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	}}
}

// NetworkExists reports whether any network carries the label.
func (c *Client) NetworkExists(ctx context.Context, label string) (bool, error) {
	out, err := c.run(ctx, "network", "ls", "--format", "{{.Name}}",
		"--filter", "label="+label)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// NetworkCreate makes a bridge network with the given labels.
func (c *Client) NetworkCreate(ctx context.Context, name string, labels ...string) error {
	args := []string{"network", "create", "--driver", "bridge"}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	args = append(args, name)
	_, err := c.run(ctx, args...)
	return err
}

// NetworkRemove deletes the named network.
func (c *Client) NetworkRemove(ctx context.Context, name string) error {
	_, err := c.run(ctx, "network", "rm", name)
	return err
}

// ContainerOptions is everything `docker run` needs for one engine.
type ContainerOptions struct {
	Name         string
	Image        string
	Labels       map[string]string
	Env          map[string]string
	Ports        map[string]string // host:container, external key
	Volumes      []string          // name:path entries
	Network      string
	NetworkAlias string
}

// ContainerState is what the feature decides on (drift checks, queries).
type (
	Container struct {
		State  ContainerState  `json:"State"`
		Config ContainerConfig `json:"Config"`
	}
	ContainerState struct {
		Running bool `json:"Running"`
	}
	ContainerConfig struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	}
)

// ErrNotFound is returned when no such container exists.
var ErrNotFound = errors.New("container not found")

// ContainerList returns container names tagged with the label.
func (c *Client) ContainerList(ctx context.Context, label string) ([]string, error) {
	out, err := c.run(ctx, "ps", "-a", "--format", "{{.Names}}", "--filter", "label="+label)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			names = append(names, l)
		}
	}
	return names, nil
}

// ContainerInspect returns the container's idle facts (running?, image,
// labels), or ErrNotFound.
func (c *Client) ContainerInspect(ctx context.Context, name string) (Container, error) {
	out, err := c.run(ctx, "inspect", name)
	if err != nil {
		if strings.Contains(string(out), "No such object") {
			return Container{}, ErrNotFound
		}
		return Container{}, err
	}
	var list []Container
	if err := json.Unmarshal(out, &list); err != nil {
		return Container{}, err
	}
	if len(list) == 0 {
		return Container{}, ErrNotFound
	}
	return list[0], nil
}

// ContainerRun starts the engine detached. Map keys and volumes are sorted so
// the argv stays deterministic. Existence/drift is the feature's call, not
// ours (we always issue the run; docker errors on duplicates).
func (c *Client) ContainerRun(ctx context.Context, o ContainerOptions) error {
	args := []string{"run", "-d", "--name", o.Name}
	for _, k := range sortedKeys(o.Labels) {
		args = append(args, "--label", k+"="+o.Labels[k])
	}
	for _, k := range sortedKeys(o.Env) {
		args = append(args, "-e", k+"="+o.Env[k])
	}
	for _, k := range sortedKeys(o.Ports) {
		args = append(args, "-p", o.Ports[k])
	}
	for _, v := range sortedStrings(o.Volumes) {
		args = append(args, "-v", v)
	}
	if o.Network != "" {
		args = append(args, "--network", o.Network)
	}
	if o.NetworkAlias != "" {
		args = append(args, "--network-alias", o.NetworkAlias)
	}
	args = append(args, o.Image)
	_, err := c.run(ctx, args...)
	return err
}

// ContainerStatus returns docker's own state word (running, exited, created,
// restarting, dead...) or ErrNotFound.
func (c *Client) ContainerStatus(ctx context.Context, name string) (string, error) {
	out, err := c.run(ctx, "inspect", "--format", "{{.State.Status}}", name)
	if err != nil {
		if strings.Contains(string(out), "No such object") {
			return "", ErrNotFound
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ContainerExec runs a command inside the container and returns combined
// output. A non-zero exit surfaces as the docker error (with stderr in the
// message); a missing container is ErrNotFound.
func (c *Client) ContainerExec(ctx context.Context, name string, args ...string) (string, error) {
	execArgs := append([]string{"exec", name}, args...)
	out, err := c.run(ctx, execArgs...)
	if err != nil {
		if strings.Contains(string(out), "No such container") {
			return "", ErrNotFound
		}
		return string(out), err
	}
	return string(out), nil
}

// ContainerStart resumes a stopped container.
func (c *Client) ContainerStart(ctx context.Context, name string) error {
	_, err := c.run(ctx, "start", name)
	return err
}

// ContainerStop halts gracefully.
func (c *Client) ContainerStop(ctx context.Context, name string) error {
	_, err := c.run(ctx, "stop", name)
	return err
}

// ContainerRestart stops then starts.
func (c *Client) ContainerRestart(ctx context.Context, name string) error {
	_, err := c.run(ctx, "restart", name)
	return err
}

// ContainerKill force-kills without removing.
func (c *Client) ContainerKill(ctx context.Context, name string) error {
	_, err := c.run(ctx, "kill", name)
	return err
}

// ImageExists reports whether the image is present locally.
func (c *Client) ImageExists(ctx context.Context, image string) (bool, error) {
	out, err := c.run(ctx, "image", "inspect", image)
	if err != nil {
		if strings.Contains(string(out), "No such image") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ImagePull fetches the image.
func (c *Client) ImagePull(ctx context.Context, image string) error {
	_, err := c.run(ctx, "pull", image)
	return err
}

// ImageRemove deletes a local image (cleanup path).
func (c *Client) ImageRemove(ctx context.Context, image string) error {
	_, err := c.run(ctx, "image", "rm", image)
	return err
}

// VolumeExists reports whether the named volume exists (exact-name match).
func (c *Client) VolumeExists(ctx context.Context, name string) (bool, error) {
	out, err := c.run(ctx, "volume", "ls", "--format", "{{.Name}}", "--filter", "name=^"+name+"$")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func (c *Client) VolumeCreate(ctx context.Context, name string) error {
	_, err := c.run(ctx, "volume", "create", name)
	return err
}

// VolumeRemove deletes the named volume (cleanup path).
func (c *Client) VolumeRemove(ctx context.Context, name string) error {
	_, err := c.run(ctx, "volume", "rm", name)
	return err
}

// ContainerRemove force-removes a container (cleanup path).
func (c *Client) ContainerRemove(ctx context.Context, name string) error {
	_, err := c.run(ctx, "rm", "-f", name)
	return err
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStrings(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
