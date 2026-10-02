// Package doorman manages the slave's engine runtime — currently only the
// per-node docker network that bounds its engines. Engine containers, health
// checks and backups rejoin this package later. Docker mechanics stay in
// internal/repositories/docker; this layer only decides.
package doorman

import (
	"context"
	"strconv"

	"noxis/internal/entity"
	"noxis/internal/repositories/docker"
)

// Networks is the docker capability doorman needs; *docker.Client satisfies
// it. Kept local — one implementation, swap-able in tests.
type Networks interface {
	NetworkExists(ctx context.Context, label string) (bool, error)
	NetworkCreate(ctx context.Context, name string, labels ...string) error
	NetworkRemove(ctx context.Context, name string) error
}

// NetworkManager owns the node's docker bridge network (noxis-<node>), the
// isolation boundary every engine container of this node joins.
type NetworkManager struct {
	name string
	d    Networks
}

func NewNetworkManager(node string) *NetworkManager {
	return &NetworkManager{name: networkName(node), d: docker.New()}
}

func networkName(node string) string { return "noxis-" + node }

func (m *NetworkManager) label() string { return "noxis.network=" + m.name }

// Reconcile makes the network exist iff the spec assigns at least one engine.
// Images need not be pulled yet — the network is the node's boundary and is
// created before any container work happens. Returns a detail string of what
// changed.
func (m *NetworkManager) Reconcile(ctx context.Context, engines []entity.Engine) (string, error) {
	exists, err := m.d.NetworkExists(ctx, m.label())
	if err != nil {
		return "", err
	}
	desired := len(engines) > 0
	switch {
	case desired && !exists:
		if err := m.d.NetworkCreate(ctx, m.name, m.label()); err != nil {
			return "", err
		}
		return "network " + m.name + " created (" + strconv.Itoa(len(engines)) + " engine(s))", nil
	case !desired && exists:
		if err := m.d.NetworkRemove(ctx, m.name); err != nil {
			return "", err
		}
		return "network " + m.name + " removed (spec empty)", nil
	case desired && exists:
		return "in sync", nil
	}
	return "", nil // no engines, no network
}
