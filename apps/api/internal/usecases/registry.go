package usecases

import "noxis/internal/modules"

// RegistryDeps is embedded by every use case that resolves an engine family:
// the registry is an explicit dependency, never read from the request context.
type RegistryDeps struct {
	Modules modules.Registry
}
