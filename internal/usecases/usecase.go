// Package usecases holds the business use cases of the noxis console, one
// file and one Serve per use case. Every use case embeds Base, which carries
// the shared machinery: the stores and engine registry it was given, the
// authorization policy and the engine lookup. Transport concerns live in
// cmd/noxis/server/handlers; a use case only speaks to repositories and
// modules, so every flow is testable without the network.
package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Base carries the shared machinery every use case embeds: the stores and
// the engine-family registry it was given.
type Base struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Grants      interfaces.GrantRepository
	Modules     map[string]interfaces.Module
}

// ErrForbidden maps to 403: editing the policy is a global-admin action.
var ErrForbidden = errors.New("forbidden: global admin required")

// RequireAdmin gates the management surfaces (connection CRUD, grants). Open
// mode (no principal) is already trusted and passes.
func (b Base) RequireAdmin(ctx context.Context) error {
	if p, ok := Principal(ctx); !ok || p.Admin {
		return nil
	}
	return ErrForbidden
}

// EffectiveLevel is the strongest level the caller's groups hold on a
// connection; "" when none matches.
func (b Base) EffectiveLevel(ctx context.Context, connection string, groups []string) (string, error) {
	grants, err := b.Grants.ListGrants(ctx, connection)
	if err != nil {
		return "", err
	}
	levels := make([]string, 0, len(grants))
	for _, g := range grants {
		if slices.Contains(groups, g.Group) {
			levels = append(levels, g.Level)
		}
	}
	return entity.StrongestLevel(levels...), nil
}

// Authorize enforces need on a connection. Open mode and global admins pass
// without a per-connection check; everyone else needs a matching grant.
func (b Base) Authorize(ctx context.Context, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := b.EffectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, need) {
		return fmt.Errorf("%w: %s requires %s on connection %q", ErrForbidden, p.Subject, need, connection)
	}
	return nil
}

// CanRead reports whether the caller may read a connection; used to hide
// connections from listings rather than failing the whole list.
func (b Base) CanRead(ctx context.Context, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := b.EffectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// readableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func (b Base) readableConnections(ctx context.Context) (map[string]bool, error) {
	list, err := b.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		c := &list[i]
		ok, err := b.CanRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		out[c.Name] = ok
	}
	return out, nil
}

// OpLevel is the level a direct SQL op requires. Everything that is not a
// read is a write (ANALYZE mutates planner statistics).
func OpLevel(operation string) string {
	if operation == constants.OpQuery {
		return entity.LevelRead
	}
	return entity.LevelWrite
}

// Raw returns the password-bearing connection for a module: a connection of
// another engine family is not addressable in this namespace. It is the
// single authorization gate every connection-scoped workflow funnels through:
// level is the access the operation needs.
func (b Base) Raw(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	c, err := b.Connections.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		if other, ok := b.moduleByDriver(c.Driver); ok {
			return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /v1/engine/%s)",
				c.Name, mod.Family(), c.Driver, other.Family())
		}
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q)",
			c.Name, mod.Family(), c.Driver)
	}
	if err := b.Authorize(ctx, name, level); err != nil {
		return nil, err
	}
	return c, nil
}

// moduleByDriver resolves the family that serves a driver through the
// registry, rather than a hardcoded driver-to-family switch.
func (b Base) moduleByDriver(driver string) (interfaces.Module, bool) {
	for _, mod := range b.Modules {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}

// Engine resolves the module named by an engine family through the registry.
// Handlers pass the request's {engine} path parameter; the instance
// workflows resolve by driver via moduleByDriver.
func (b Base) Engine(name string) (interfaces.Module, error) {
	mod, ok := b.Modules[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
	}
	return mod, nil
}

// EngineSQL, EngineMongo and EngineMemcache resolve an engine family and
// narrow it to the capability the caller needs; a family that does not serve
// it is unsupported (404).
func (b Base) EngineSQL(name string) (interfaces.SQLModule, error) {
	mod, err := b.Engine(name)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func (b Base) EngineMongo(name string) (interfaces.MongoModule, error) {
	mod, err := b.Engine(name)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func (b Base) EngineMemcache(name string) (interfaces.MemcacheModule, error) {
	mod, err := b.Engine(name)
	if err != nil {
		return nil, err
	}
	return memcacheOf(mod)
}

func sqlOf(mod interfaces.Module) (interfaces.SQLModule, error) {
	m, ok := mod.(interfaces.SQLModule)
	if !ok {
		return nil, unsupported(mod, "sql operations")
	}
	return m, nil
}

func mongoOf(mod interfaces.Module) (interfaces.MongoModule, error) {
	m, ok := mod.(interfaces.MongoModule)
	if !ok {
		return nil, unsupported(mod, "mongodb operations")
	}
	return m, nil
}

func memcacheOf(mod interfaces.Module) (interfaces.MemcacheModule, error) {
	m, ok := mod.(interfaces.MemcacheModule)
	if !ok {
		return nil, unsupported(mod, "memory-cache operations")
	}
	return m, nil
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: engine %q does not serve %s", interfaces.ErrUnsupported, mod.Family(), what)
}
