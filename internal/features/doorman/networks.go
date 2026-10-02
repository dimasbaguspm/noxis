// Package doorman manages the slave's engine runtime — currently only the
// per-node docker network that bounds its engines. Engine containers, health
// checks and backups rejoin this package later.
package doorman

import (
	"context"
	"os/exec"
	"strconv"
	"strings"

	"noxis/internal/entity"
)

// Runner executes docker commands; swapped for a fake in tests.
type Runner func(ctx context.Context, args ...string) ([]byte, error)

// NetworkManager owns the node's docker bridge network (noxis-<node>), the
// isolation boundary every engine container of this node joins.
type NetworkManager struct {
	name string
	run  Runner
}

func NewNetworkManager(node string) *NetworkManager {
	return &NetworkManager{
		name: networkName(node),
		run: func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		},
	}
}

func networkName(node string) string { return "noxis-" + node }

// Reconcile makes the network exist iff the spec assigns at least one engine.
// Images need not be pulled yet — the network is the node's boundary and is
// created before any container work happens. Returns a detail string of what
// changed.
func (m *NetworkManager) Reconcile(ctx context.Context, engines []entity.Engine) (string, error) {
	exists, err := m.exists(ctx)
	if err != nil {
		return "", err
	}
	desired := len(engines) > 0
	switch {
	case desired && !exists:
		if _, err := m.run(ctx, "network", "create", "--driver", "bridge",
			"--label", "noxis.network="+m.name, m.name); err != nil {
			return "", err
		}
		return "network " + m.name + " created (" + strconv.Itoa(len(engines)) + " engine(s))", nil
	case !desired && exists:
		if _, err := m.run(ctx, "network", "rm", m.name); err != nil {
			return "", err
		}
		return "network " + m.name + " removed (spec empty)", nil
	case desired && exists:
		return "in sync", nil
	}
	return "", nil // no engines, no network
}

func (m *NetworkManager) exists(ctx context.Context) (bool, error) {
	out, err := m.run(ctx, "network", "ls", "--format", "{{.Name}}",
		"--filter", "label=noxis.network="+m.name)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), m.name), nil
}
