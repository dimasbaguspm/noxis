// Package deps is the runtime dependency container of the server. Deps holds
// the sources the initiator provides — repositories and the engine registry
// — and Init binds the use cases that consume them into a Uses. Deps itself
// never lists a use case.
package deps

import (
	"context"
	"log/slog"

	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
)

// Deps is what the initiator provides: the repositories, the registry (which
// carries the auth service) and the transport values. WithDeps fills a use case
// from it by field type.
type Deps struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
	Users       interfaces.UserRepository
	Modules     modules.Registry
	Log         *slog.Logger
	Auth        config.AuthInfo
}

type usesKey struct{}

// Seed returns a context carrying the boot-time uses.
func Seed(ctx context.Context, u Uses) context.Context {
	return context.WithValue(ctx, usesKey{}, u)
}

// From returns the uses carried by the context. It panics when the context
// was not seeded: a handler without uses is a wiring bug and must surface at
// the first request, not quietly act on zero values.
func From(ctx context.Context) Uses {
	u, ok := ctx.Value(usesKey{}).(Uses)
	if !ok {
		panic("deps.From: uses missing from context (seed with deps.Seed at boot)")
	}
	return u
}
