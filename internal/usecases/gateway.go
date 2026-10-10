package usecases

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The workflow seams handlers depend on. Each gateway implements its seam by
// delegating to the individual use cases; a handler never sees a bigger
// surface than the routes it serves.
type (
	// EngineWorkflows resolves the engine a route's {engine} parameter names.
	EngineWorkflows interface {
		Engine(name string) (interfaces.Module, error)
		EngineSQL(name string) (interfaces.SQLModule, error)
		EngineMongo(name string) (interfaces.MongoModule, error)
		EngineMemcache(name string) (interfaces.MemcacheModule, error)
	}

	// ChangesWorkflows is the SQL change queue.
	ChangesWorkflows interface {
		SubmitChange(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (entity.Change, error)
		Approve(ctx context.Context, id string) (entity.Change, error)
		Schedule(ctx context.Context, id string, at int64) (entity.Change, error)
		Apply(ctx context.Context, sq interfaces.SQLModule, id string) (entity.Change, error)
		Reject(ctx context.Context, id string) (entity.Change, error)
		SQLChanges(ctx context.Context) ([]entity.Change, error)
		SQLChange(ctx context.Context, id string) (*entity.Change, error)
		Tick(ctx context.Context) (TickResult, error)
	}

	// ConnectionsWorkflows is a family's connection registry, its schema
	// browser and the direct SQL operations. Raw is the shared authorization
	// gate for the engine-scoped routes.
	ConnectionsWorkflows interface {
		ConnectionList(ctx context.Context, mod interfaces.Module) ([]entity.Connection, error)
		Connection(ctx context.Context, mod interfaces.Module, name string) (entity.Connection, error)
		Raw(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error)
		SaveConnection(ctx context.Context, mod interfaces.Module, c entity.Connection) (entity.Connection, error)
		DeleteConnection(ctx context.Context, mod interfaces.Module, name string) error
		TestConnection(ctx context.Context, mod interfaces.Module, name string) error
		Schema(ctx context.Context, sq interfaces.SQLModule, name string) ([]interfaces.Table, error)
		Rows(ctx context.Context, sq interfaces.SQLModule, name, schema, table string, limit int) (interfaces.Result, error)
		Op(ctx context.Context, sq interfaces.SQLModule, name, operation, sqlText, table string) (interfaces.Result, error)
	}

	// MongoWorkflows is the MongoDB review queue and the direct read routes.
	MongoWorkflows interface {
		SubmitMongoChange(ctx context.Context, m interfaces.MongoModule, conn, notes string, req interfaces.MongoRequest) (entity.Change, error)
		MongoChange(ctx context.Context, id string) (*entity.Change, error)
		ApplyMongo(ctx context.Context, m interfaces.MongoModule, id string) (entity.Change, error)
		MongoChanges(ctx context.Context) ([]entity.Change, error)
	}

	// InstanceWorkflows is the cross-family instance view.
	InstanceWorkflows interface {
		Instance(ctx context.Context, id string) (entity.Connection, error)
		Instances(ctx context.Context, name, driver, host string) ([]entity.Connection, error)
		RotatePassword(ctx context.Context, id, password string) error
	}

	// GrantWorkflows is the per-connection RBAC policy, admin only.
	GrantWorkflows interface {
		ListGrants(ctx context.Context, mod interfaces.Module, name string) ([]entity.Grant, error)
		SetGrant(ctx context.Context, mod interfaces.Module, name, group, level string) (entity.Grant, error)
		DeleteGrant(ctx context.Context, mod interfaces.Module, name, group string) error
		Groups(ctx context.Context) ([]string, error)
	}

	// MeWorkflows is the caller's own per-connection levels.
	MeWorkflows interface {
		ConnectionLevels(ctx context.Context) (map[string]string, error)
	}
)

// The gateways: one struct per seam, delegating to the individual use cases
// over a shared Base.
type (
	Engines     struct{ Base }
	Changes     struct{ Base }
	Connections struct{ Base }
	Mongo       struct{ Base }
	Instances   struct{ Base }
	Grants      struct{ Base }
	Me          struct{ Base }
)

func (g Engines) Engine(name string) (interfaces.Module, error)       { return g.Base.Engine(name) }
func (g Engines) EngineSQL(name string) (interfaces.SQLModule, error) { return g.Base.EngineSQL(name) }
func (g Engines) EngineMongo(name string) (interfaces.MongoModule, error) {
	return g.Base.EngineMongo(name)
}
func (g Engines) EngineMemcache(name string) (interfaces.MemcacheModule, error) {
	return g.Base.EngineMemcache(name)
}

func (g Changes) SubmitChange(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (entity.Change, error) {
	return (SubmitChange(g)).Serve(ctx, sq, ch)
}
func (g Changes) Approve(ctx context.Context, id string) (entity.Change, error) {
	return (Approve(g)).Serve(ctx, id)
}
func (g Changes) Schedule(ctx context.Context, id string, at int64) (entity.Change, error) {
	return (Schedule(g)).Serve(ctx, id, at)
}
func (g Changes) Apply(ctx context.Context, sq interfaces.SQLModule, id string) (entity.Change, error) {
	return (Apply(g)).Serve(ctx, sq, id)
}
func (g Changes) Reject(ctx context.Context, id string) (entity.Change, error) {
	return (Reject(g)).Serve(ctx, id)
}
func (g Changes) SQLChanges(ctx context.Context) ([]entity.Change, error) {
	return (SQLChanges(g)).Serve(ctx)
}
func (g Changes) SQLChange(ctx context.Context, id string) (*entity.Change, error) {
	return (SQLChange(g)).Serve(ctx, id)
}
func (g Changes) Tick(ctx context.Context) (TickResult, error) { return (Tick(g)).Serve(ctx) }

func (g Connections) ConnectionList(ctx context.Context, mod interfaces.Module) ([]entity.Connection, error) {
	return (ConnectionList(g)).Serve(ctx, mod)
}
func (g Connections) Connection(ctx context.Context, mod interfaces.Module, name string) (entity.Connection, error) {
	return (Connection(g)).Serve(ctx, mod, name)
}
func (g Connections) Raw(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	return g.Base.Raw(ctx, mod, name, level)
}
func (g Connections) SaveConnection(ctx context.Context, mod interfaces.Module, c entity.Connection) (entity.Connection, error) {
	return (SaveConnection(g)).Serve(ctx, mod, c)
}
func (g Connections) DeleteConnection(ctx context.Context, mod interfaces.Module, name string) error {
	return (DeleteConnection(g)).Serve(ctx, mod, name)
}
func (g Connections) TestConnection(ctx context.Context, mod interfaces.Module, name string) error {
	return (TestConnection(g)).Serve(ctx, mod, name)
}
func (g Connections) Schema(ctx context.Context, sq interfaces.SQLModule, name string) ([]interfaces.Table, error) {
	return (Schema(g)).Serve(ctx, sq, name)
}
func (g Connections) Rows(ctx context.Context, sq interfaces.SQLModule, name, schema, table string, limit int) (interfaces.Result, error) {
	return (Rows(g)).Serve(ctx, sq, name, schema, table, limit)
}
func (g Connections) Op(ctx context.Context, sq interfaces.SQLModule, name, operation, sqlText, table string) (interfaces.Result, error) {
	return (Op(g)).Serve(ctx, sq, name, operation, sqlText, table)
}

func (g Mongo) SubmitMongoChange(ctx context.Context, m interfaces.MongoModule, conn, notes string, req interfaces.MongoRequest) (entity.Change, error) {
	return (SubmitMongoChange(g)).Serve(ctx, m, conn, notes, req)
}
func (g Mongo) MongoChange(ctx context.Context, id string) (*entity.Change, error) {
	return (MongoChange(g)).Serve(ctx, id)
}
func (g Mongo) ApplyMongo(ctx context.Context, m interfaces.MongoModule, id string) (entity.Change, error) {
	return (ApplyMongo(g)).Serve(ctx, m, id)
}
func (g Mongo) MongoChanges(ctx context.Context) ([]entity.Change, error) {
	return (MongoChanges(g)).Serve(ctx)
}

func (g Instances) Instance(ctx context.Context, id string) (entity.Connection, error) {
	return (Instance(g)).Serve(ctx, id)
}
func (g Instances) Instances(ctx context.Context, name, driver, host string) ([]entity.Connection, error) {
	return (ListInstances(g)).Serve(ctx, name, driver, host)
}
func (g Instances) RotatePassword(ctx context.Context, id, password string) error {
	return (RotatePassword(g)).Serve(ctx, id, password)
}

func (g Grants) ListGrants(ctx context.Context, mod interfaces.Module, name string) ([]entity.Grant, error) {
	return (ListGrants(g)).Serve(ctx, mod, name)
}
func (g Grants) SetGrant(ctx context.Context, mod interfaces.Module, name, group, level string) (entity.Grant, error) {
	return (SetGrant(g)).Serve(ctx, mod, name, group, level)
}
func (g Grants) DeleteGrant(ctx context.Context, mod interfaces.Module, name, group string) error {
	return (DeleteGrant(g)).Serve(ctx, mod, name, group)
}
func (g Grants) Groups(ctx context.Context) ([]string, error) { return (Groups(g)).Serve(ctx) }

func (g Me) ConnectionLevels(ctx context.Context) (map[string]string, error) {
	return (ConnectionLevels(g)).Serve(ctx)
}

// Satisfiers: the gateways implement their seams.
var (
	_ EngineWorkflows      = (*Engines)(nil)
	_ ChangesWorkflows     = (*Changes)(nil)
	_ ConnectionsWorkflows = (*Connections)(nil)
	_ MongoWorkflows       = (*Mongo)(nil)
	_ InstanceWorkflows    = (*Instances)(nil)
	_ GrantWorkflows       = (*Grants)(nil)
	_ MeWorkflows          = (*Me)(nil)
)
