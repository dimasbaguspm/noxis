package usecases

import (
	"context"
	"log/slog"

	"noxis/internal/common/config"
	cache "noxis/internal/repositories/in-memory-cache"
)

// Deps is everything a transport handler reads through the request context:
// the use cases plus the transport-global values (public OIDC config, tick
// key and the shared cache). It is assembled once at boot (server.Run) and
// seeded into the base request context; handlers consume it with From. It is
// never rebuilt per request.
type Deps struct {
	Engines     EngineWorkflows
	Changes     ChangesWorkflows
	Connections ConnectionsWorkflows
	Mongo       MongoWorkflows
	Instances   InstanceWorkflows
	Grants      GrantWorkflows
	Me          MeWorkflows
	Log         *slog.Logger
	Cache       *cache.Cache
	Auth        config.AuthInfo
}

type depsKey struct{}

// WithDeps returns a context carrying the boot-time deps.
func WithDeps(ctx context.Context, deps Deps) context.Context {
	return context.WithValue(ctx, depsKey{}, deps)
}

// From returns the deps carried by the context. It panics when the context
// was not built by WithDeps: a handler without deps is a wiring bug and must
// surface at the first request, not quietly nil-dereference.
func From(ctx context.Context) Deps {
	deps, ok := ctx.Value(depsKey{}).(Deps)
	if !ok {
		panic("usecases.From: deps missing from context (seed with WithDeps in server.Run)")
	}
	return deps
}
