// Package application holds the use cases of the noise console: the change
// lifecycle, connection registry, grants policy and cross-family instance
// lookup. Transport concerns (HTTP, swagger) live in cmd/noxis/server/api; an
// App here only speaks to repositories and the engine-family registry it is
// given, so every workflow is testable without the network.
package application

import (
	"context"
	"log/slog"

	"noxis/internal/common/logging"
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
	Log         *slog.Logger
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

// span opens the span for the calling method (the name derives from the
// caller) and logs its outcome.
func (a *App) span(ctx context.Context) (ctxOut context.Context, end func(error)) {
	return a.spanNamed(ctx, logging.CallerName(1))
}

func (a *App) spanNamed(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, endSpan := logging.StartSpan(ctx, name)
	return ctxOut, func(err error) {
		if err != nil {
			a.Log.WarnContext(ctxOut, name, "error", err.Error())
		} else {
			a.Log.InfoContext(ctxOut, name)
		}
		endSpan(err)
	}
}