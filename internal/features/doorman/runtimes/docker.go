package runtimes

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"noxis/internal/entity"
)

// Exec shells out to the docker CLI: zero SDK dependency, every action
// readable as a docker command, and swappable in tests.
var Exec = func(_ context.Context, args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	out = []byte(strings.TrimSpace(string(out)))
	if err != nil && len(out) > 0 {
		err = fmt.Errorf("%v: %s", err, string(out))
	}
	return string(out), err
}

type Observed struct {
	State   string
	Desired string // the noxis.desired label: what this container was created to be
}

// List returns this node's noxis-managed containers keyed by container name.
// Node-scoped by label: on a shared docker host one slave never touches
// another's containers. Names come from `docker ps`; labels come from
// `docker inspect`, whose Config.Labels is a real map (ps --format exposes
// labels as a raw slice in current docker).
func List(ctx context.Context, node string) (map[string]Observed, error) {
	ids, err := Exec(ctx, "ps", "-aq",
		"--filter", "label=noxis=true",
		"--filter", "label=noxis.node="+node)
	if err != nil {
		return nil, err
	}
	found := map[string]Observed{}
	if strings.TrimSpace(ids) == "" {
		return found, nil
	}
	out, err := Exec(ctx, append([]string{"inspect",
		"--format", "{{.Name}}\t{{.State.Status}}\t{{index .Config.Labels \"noxis.desired\"}}"},
		strings.Fields(ids)...)...)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) == 3 {
			found[strings.TrimPrefix(parts[0], "/")] = Observed{State: parts[1], Desired: parts[2]}
		}
	}
	return found, nil
}

// Run creates and starts an engine container, stamping the exact desired
// shape into noxis.desired so later reconciles detect drift label-only.
func Run(ctx context.Context, node string, e entity.Engine) error {
	d, ok := Lookup(e.Type)
	if !ok {
		return fmt.Errorf("unknown engine type %q", e.Type)
	}
	args := []string{"run", "-d",
		"--name", ContainerName(node, e.Name),
		"--label", "noxis=true",
		"--label", "noxis.node=" + node,
		"--restart", "unless-stopped",
	}
	for host, container := range mergedPorts(e, d) {
		args = append(args, "-p", host+":"+container)
	}
	for _, kv := range sortedEnv(mergedEnv(e, d)) {
		args = append(args, "-e", kv[0]+"="+kv[1])
	}
	desired, err := DesiredLabel(e)
	if err != nil {
		return err
	}
	args = append(args, "--label", "noxis.desired="+desired)
	args = append(args, d.image(e.Version))
	_, err = Exec(ctx, args...)
	return err
}

func Remove(ctx context.Context, name string) error {
	_, err := Exec(ctx, "rm", "-f", name)
	return err
}

func Start(ctx context.Context, name string) error {
	_, err := Exec(ctx, "start", name)
	return err
}

func sortedEnv(env map[string]string) [][2]string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][2]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, [2]string{k, env[k]})
	}
	return out
}
