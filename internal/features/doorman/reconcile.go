package doorman

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"noxis/internal/core"
)

// Reconciler converges a node's slice of desired state against the local
// docker daemon. Pure executor: it makes no decisions, only diff + apply.
type Reconciler struct {
	Node string
}

func NewReconciler(node string) *Reconciler { return &Reconciler{Node: node} }

func ContainerName(node, engine string) string {
	return fmt.Sprintf("noxis-%s-%s", node, engine)
}

// Reconcile diffs desired engines against running containers and applies.
// Returns a human-readable summary of what changed.
func (r *Reconciler) Reconcile(ctx context.Context, engines []*core.EngineSpec) (string, error) {
	desired := map[string]*core.EngineSpec{}
	for _, e := range engines {
		if _, ok := Lookup(e.Type); !ok {
			return "", fmt.Errorf("unknown engine type %q", e.Type)
		}
		desired[r.container(e)] = e
	}

	actual, err := listContainers(ctx, r.Node)
	if err != nil {
		return "", err
	}

	var lines []string
	for name, e := range desired {
		d, _ := Lookup(e.Type)
		image := d.image(e.Version)
		obs, exists := actual[name]
		switch {
		case !exists:
			if err := runContainer(ctx, r.nodeName(), e); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("started %s (%s)", e.Name, image))
		case obs.image != image:
			if err := removeContainer(ctx, name); err != nil {
				return "", err
			}
			if err := runContainer(ctx, r.nodeName(), e); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("upgraded %s (%s)", e.Name, image))
		case obs.state != "running":
			if _, err := dockerCmd(ctx, "start", name); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("started %s", e.Name))
		}
	}

	for name := range actual {
		if _, want := desired[name]; !want {
			if err := removeContainer(ctx, name); err != nil {
				return "", err
			}
			lines = append(lines, "removed "+name)
		}
	}

	if len(lines) == 0 {
		return "", nil // in sync
	}
	sort.Strings(lines)
	return strings.Join(lines, "; "), nil
}

func (r *Reconciler) nodeName() string { return r.Node }

func (r *Reconciler) container(e *core.EngineSpec) string {
	return ContainerName(r.Node, e.Name)
}

func runContainer(ctx context.Context, node string, e *core.EngineSpec) error {
	d, _ := Lookup(e.Type)
	args := []string{"run", "-d",
		"--name", ContainerName(node, e.Name),
		"--label", "noxis=true",
		"--label", "noxis.node=" + node,
		"--restart", "unless-stopped",
	}
	ports := e.Ports
	if len(ports) == 0 {
		ports = d.Ports
	}
	for host, container := range ports {
		args = append(args, "-p", host+":"+container)
	}
	env := map[string]string{}
	for k, v := range d.Env {
		env[k] = v
	}
	for k, v := range e.Env {
		env[k] = v
	}
	for _, kv := range sortedEnv(env) {
		args = append(args, "-e", kv[0]+"="+kv[1])
	}
	args = append(args, d.image(e.Version))
	_, err := dockerCmd(ctx, args...)
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