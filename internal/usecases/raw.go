package usecases

import (
	"context"
	"fmt"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
)

// rawConn returns the password-bearing connection for a module: a connection of
// another engine family is not addressable in this namespace. It is the
// single authorization gate every connection-scoped workflow funnels through:
// level is the access the operation needs.
func rawConn(ctx context.Context, conns interfaces.ConnectionRepository, grants interfaces.GrantRepository, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	c, err := conns.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		if other, ok := modules.RegistryFrom(ctx).Serves(c.Driver); ok {
			return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /v1/engine/%s)",
				c.Name, mod.Family(), c.Driver, other.Family())
		}
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q)",
			c.Name, mod.Family(), c.Driver)
	}
	if err := Authorize(ctx, grants, name, level); err != nil {
		return nil, err
	}
	return c, nil
}
