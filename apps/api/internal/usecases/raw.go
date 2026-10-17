package usecases

import (
	"context"
	"fmt"

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
