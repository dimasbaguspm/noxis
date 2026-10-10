package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Instance is the "Instance" use case.
type Instance struct{ Base }

// Serve runs it.
func (uc Instance) Serve(ctx context.Context, id string) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := uc.Connections.GetByID(ctx, id)
	if err != nil {
		return entity.Connection{}, err
	}
	mod, ok := uc.moduleByDriver(c.Driver)
	if !ok {
		return entity.Connection{}, interfaces.ErrUnsupported
	}
	readable, err := uc.CanRead(ctx, c.Name)
	if err != nil {
		return entity.Connection{}, err
	}
	if !readable {
		return entity.Connection{}, interfaces.ErrNotFound
	}
	c.Family = mod.Family()
	return c.Redacted(), nil
}
