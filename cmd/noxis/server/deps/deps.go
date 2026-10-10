// Package deps is the runtime dependency container of the server: the
// repositories, the engine registry and the bound use cases a request can
// call. The initiator lives here — in cmd, not in the use cases: Init binds
// every use case once, WithDeps fills a use case's fields from the sources.
package deps

import (
	"context"
	"log/slog"

	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories/in-memory-cache"
	"noxis/internal/usecases"
)

// Deps is what a request carries: the runtime sources (repositories, engine
// registry, transport values) plus the use cases bound to them at boot.
type Deps struct {
	// Sources: what the initiator provides and WithDeps injects from.
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Grants      interfaces.GrantRepository
	Modules     modules.Registry
	Log         *slog.Logger
	Cache       *cache.Cache
	Auth        config.AuthInfo

	// Bound use cases (filled by Init).
	SubmitChange     usecases.SubmitChange
	Approve          usecases.Approve
	Schedule         usecases.Schedule
	Apply            usecases.Apply
	Reject           usecases.Reject
	SQLChanges       usecases.SQLChanges
	SQLChange        usecases.SQLChange
	Tick             usecases.Tick
	ConnectionList   usecases.ConnectionList
	Connection       usecases.Connection
	Raw              usecases.Raw
	SaveConnection   usecases.SaveConnection
	DeleteConnection usecases.DeleteConnection
	TestConnection   usecases.TestConnection
	Schema           usecases.Schema
	Rows             usecases.Rows
	Op               usecases.Op
	SubmitMongo      usecases.SubmitMongoChange
	MongoChange      usecases.MongoChange
	ApplyMongo       usecases.ApplyMongo
	MongoChanges     usecases.MongoChanges
	Instance         usecases.Instance
	Instances        usecases.ListInstances
	RotatePassword   usecases.RotatePassword
	ListGrants       usecases.ListGrants
	SetGrant         usecases.SetGrant
	DeleteGrant      usecases.DeleteGrant
	Groups           usecases.Groups
	ConnectionLevels usecases.ConnectionLevels
}

type depsKey struct{}

// Seed returns a context carrying the boot-time deps.
func Seed(ctx context.Context, d Deps) context.Context {
	return context.WithValue(ctx, depsKey{}, d)
}

// From returns the deps carried by the context. It panics when the context
// was not seeded: a handler without deps is a wiring bug and must surface at
// the first request, not quietly act on zero values.
func From(ctx context.Context) Deps {
	d, ok := ctx.Value(depsKey{}).(Deps)
	if !ok {
		panic("deps.From: deps missing from context (seed with deps.Seed at boot)")
	}
	return d
}
