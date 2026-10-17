package usecases

import (
	"context"
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// RawConn returns the password-bearing connection for a module: a connection
// of another engine family is not addressable in this namespace. It is the
// single authorization gate every connection-scoped workflow funnels through.
// The repositories and the registry ride in the receiver, so this stays a
// method rather than a function taking three dependencies.
func (d RegistryDeps) RawConn(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	c, err := d.Connections.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		if other, ok := d.Modules.Serves(c.Driver); ok {
			return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /external/v1/engine/%s)",
				c.Name, mod.Family(), c.Driver, other.Family())
		}
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q)",
			c.Name, mod.Family(), c.Driver)
	}
	if err := d.Authorize(ctx, name, level); err != nil {
		return nil, err
	}
	return c, nil
}

// SqlChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
func (d RegistryDeps) SQLChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return nil, fmt.Errorf("change %s is %s, not a sql change", id, ch.Kind)
	}
	if err := d.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// MongoChange loads a change and insists it is a MongoDB request, so the SQL
// queue cannot be addressed through the MongoDB flow.
func (d RegistryDeps) MongoChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	if err := d.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}
