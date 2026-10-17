// Package repositories aggregates the persistence seams so a caller can depend
// on one value instead of enumerating each repository. The concrete store lives
// in the sqlite subpackage.
package repositories

import "noxis/internal/interfaces"

// Registry bundles the repositories the application reads through. It is
// embedded in use cases (via usecases.RegistryDeps) so a use case reaches its
// repositories as promoted fields: uc.Connections, uc.Changes, uc.Users.
type Registry struct {
	Connections    interfaces.ConnectionRepository
	Changes        interfaces.ChangeRepository
	Permissions    interfaces.PermissionRepository
	Users          interfaces.UserRepository
	AccessEvents   interfaces.AccessEventRepository
	AccessRequests interfaces.AccessRequestRepository
}
