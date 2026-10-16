package usecases

import (
	"noxis/internal/modules"
	"noxis/internal/repositories"
)

// RegistryDeps is embedded by every use case. It promotes the repositories
// (Connections, Changes, Permissions, Users) and the module registry (Modules),
// so a use case depends on one value instead of enumerating its dependencies,
// and none of them are read from the request context.
type RegistryDeps struct {
	repositories.Registry
	Modules modules.Registry
}
