package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ConnectionList lists a family's connections, gated by readability and the
// family's driver set.
func (a *App) ConnectionList(ctx context.Context, mod interfaces.Module) (out []entity.Connection, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	list, err := a.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		if !mod.Supports(c.Driver) {
			continue
		}
		ok, err := a.CanRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, c.Redacted())
		}
	}
	return out, nil
}

// Connection returns one connection of a family, redacted.
func (a *App) Connection(ctx context.Context, mod interfaces.Module, name string) (out entity.Connection, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	c, err := a.Raw(ctx, mod, name, entity.LevelRead)
	if err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}

// Raw returns the password-bearing connection for a module: a connection of
// another engine family is not addressable in this namespace. It is also the
// single authorization gate every connection-scoped workflow funnels through:
// level is the access the operation needs.
func (a *App) Raw(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	c, err := a.Connections.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		if other, ok := a.moduleByDriver(c.Driver); ok {
			return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /v1/engine/%s)",
				c.Name, mod.Family(), c.Driver, other.Family())
		}
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q)",
			c.Name, mod.Family(), c.Driver)
	}
	if err := a.Authorize(ctx, name, level); err != nil {
		return nil, err
	}
	return c, nil
}

// SaveConnection upserts; an update that omits the password keeps the stored
// one (the console never sees it).
func (a *App) SaveConnection(ctx context.Context, mod interfaces.Module, c entity.Connection) (out entity.Connection, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return entity.Connection{}, err
	}
	if !mod.Supports(c.Driver) {
		return entity.Connection{}, fmt.Errorf("driver %q is not a %s driver", c.Driver, mod.Family())
	}
	if err := c.Validate(); err != nil {
		return entity.Connection{}, err
	}
	existing, err := a.Connections.Get(ctx, c.Name)
	switch {
	case err == nil:
		// identity is fixed at registration: a different driver, host,
		// database or user is a different instance, not an edit. The password
		// rotates through its own action (RotatePassword).
		if c.Driver != existing.Driver || c.Host != existing.Host ||
			c.Database != existing.Database || c.User != existing.User {
			return entity.Connection{}, errors.New(
				"driver, host, database and user cannot be changed after registration; " +
					"rotate the password or register a new connection")
		}
		if c.Password == "" {
			c.Password = existing.Password
		}
		c.CreatedAt = existing.CreatedAt
		c.CreatedBy = existing.CreatedBy
		c.ID = existing.ID
	case errors.Is(err, interfaces.ErrNotFound):
		c.CreatedAt = time.Now().Unix()
		c.CreatedBy = actor(ctx)
		c.ID = idgen.UUID()
	default:
		return entity.Connection{}, err
	}
	if err := a.Connections.Save(ctx, c); err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}

// DeleteConnection removes a connection and its changes after checking the
// connection belongs to this family, so one namespace cannot delete another
// family's server by name.
func (a *App) DeleteConnection(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return err
	}
	if _, err := a.Raw(ctx, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return a.Connections.Delete(ctx, name)
}

// TestConnection pings a connection through its module.
func (a *App) TestConnection(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	c, err := a.Raw(ctx, mod, name, entity.LevelRead)
	if err != nil {
		return err
	}
	return mod.Ping(ctx, *c)
}

// Schema introspects a SQL connection's tables and columns.
func (a *App) Schema(ctx context.Context, sq interfaces.SQLModule, name string) (out []interfaces.Table, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	c, err := a.Raw(ctx, sq, name, entity.LevelRead)
	if err != nil {
		return nil, err
	}
	return sq.Tables(ctx, *c)
}

// Rows previews a table's rows.
func (a *App) Rows(ctx context.Context, sq interfaces.SQLModule, name, schema, table string, limit int) (out interfaces.Result, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	c, err := a.Raw(ctx, sq, name, entity.LevelRead)
	if err != nil {
		return interfaces.Result{}, err
	}
	return sq.Rows(ctx, *c, schema, table, limit)
}

// Op dispatches one direct SQL operation.
func (a *App) Op(ctx context.Context, sq interfaces.SQLModule, name, operation, sqlText, table string) (out interfaces.Result, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	c, err := a.Raw(ctx, sq, name, OpLevel(operation))
	if err != nil {
		return interfaces.Result{}, err
	}
	switch operation {
	case constants.OpQuery:
		return sq.Query(ctx, *c, sqlText)
	case constants.OpAlter:
		return sq.Alter(ctx, *c, sqlText)
	case constants.OpRollback:
		return sq.DryRun(ctx, *c, sqlText)
	case constants.OpAnalyze:
		return sq.Analyze(ctx, *c, table)
	}
	return interfaces.Result{}, fmt.Errorf("unknown op %q", operation)
}