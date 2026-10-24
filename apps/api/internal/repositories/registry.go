// Package repositories aggregates the persistence seams so a caller can depend on one value instead of enumerating each repository.
package repositories

import "noxis/internal/interfaces"

// Registry bundles repositories promoted through usecases.RegistryDeps.
type Registry struct {
	Connections    interfaces.ConnectionRepository
	Changes        interfaces.ChangeRepository
	Permissions    interfaces.PermissionRepository
	Users          interfaces.UserRepository
	AccessEvents   interfaces.AccessEventRepository
	AccessRequests interfaces.AccessRequestRepository
}

// Store is the one persistence implementation every repository is read from.
type Store interface {
	interfaces.ConnectionRepository
	interfaces.ChangeRepository
	interfaces.PermissionRepository
	interfaces.UserRepository
	interfaces.AccessEventRepository
	interfaces.AccessRequestRepository
}

// Default builds the registry from the process store.
func Default(store Store) Registry {
	return Registry{
		Connections:    store,
		Changes:        store,
		Permissions:    store,
		Users:          store,
		AccessEvents:   store,
		AccessRequests: store,
	}
}
