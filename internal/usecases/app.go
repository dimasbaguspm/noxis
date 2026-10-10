// Package usecases holds the use cases of the noise console: the change
// lifecycle, connection registry, grants policy and cross-family instance
// lookup. Transport concerns (HTTP, swagger) live in cmd/noxis/server/api; an
// App here only speaks to repositories and the engine-family registry it is
// given, so every workflow is testable without the network.
package usecases

import (
	"noxis/internal/interfaces"
)

// App is the composed application: the shared stores plus the engine-family
// registry. Handlers resolve the engine from the request and pass the module
// into the workflows; the tick resolves it from a connection's driver.
type App struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Grants      interfaces.GrantRepository
	Modules     map[string]interfaces.Module
}

// moduleByDriver resolves the family that serves a driver through the
// registry, rather than a hardcoded driver-to-family switch.
func (a *App) moduleByDriver(driver string) (interfaces.Module, bool) {
	for _, mod := range a.Modules {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}
