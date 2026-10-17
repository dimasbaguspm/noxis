package usecases

import (
	"time"

	"noxis/internal/constants"
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
	// AccessTTL is how long a per-connection grant lasts. Zero falls back to
	// constants.AccessGrantTTL, so a hand-built dep in a test still works.
	AccessTTL time.Duration
}

// GrantTTL is the configured grant lifetime, or the 120-day default.
func (d RegistryDeps) GrantTTL() time.Duration {
	if d.AccessTTL <= 0 {
		return constants.AccessGrantTTL
	}
	return d.AccessTTL
}
