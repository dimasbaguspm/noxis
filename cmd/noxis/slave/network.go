// The slave's node isolation: a per-node docker bridge network
// (noxis-<node>) that bounds the node's engine containers. This is worker
// infrastructure, not a feature module — docker mechanics stay in
// internal/repositories/docker; this layer only decides.
package slave

import (
	"context"
	"strconv"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/docker"
)

// Domain conventions for the node's isolation resources, owned here.
const (
	networkPrefix = "noxis-"        // docker resource name prefix
	networkLabel  = "noxis.network" // label key pinning a network to its node
)

// NetworkManager owns the node's docker bridge network (noxis-<node>).
type NetworkManager struct {
	name string
	d    interfaces.Docker
}

func NewNetworkManager(node string) *NetworkManager {
	return &NetworkManager{name: networkName(node), d: docker.New()}
}

func networkName(node string) string { return networkPrefix + node }

func (m *NetworkManager) label() string { return networkLabel + "=" + m.name }

// NetworkName is the docker network name for this manager's node.
func (m *NetworkManager) NetworkName() string { return m.name }

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
