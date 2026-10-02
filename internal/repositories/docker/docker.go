// Package docker is the docker CLI wrapper: every docker command the app
// issues lives here so feature code never builds shell invocations. Runs via
// the docker socket (CLI binary inside containers).
package docker

import (
	"context"
	"os/exec"
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
