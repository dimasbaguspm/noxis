package usecases

import (
	"context"
	"errors"
	"fmt"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"time"
)

// SaveConnection is the "SaveConnection" use case.
type SaveConnection struct {
	Connections interfaces.ConnectionRepository
}

// Serve runs it.
func (uc SaveConnection) Serve(ctx context.Context, mod interfaces.Module, c entity.Connection) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return entity.Connection{}, err
	}
	if !mod.Supports(c.Driver) {
		return entity.Connection{}, fmt.Errorf("driver %q is not a %s driver", c.Driver, mod.Family())
	}
	if err := c.Validate(); err != nil {
		return entity.Connection{}, err
	}
	existing, err := uc.Connections.Get(ctx, c.Name)
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
	if err := uc.Connections.Save(ctx, c); err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}
