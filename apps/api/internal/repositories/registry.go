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
