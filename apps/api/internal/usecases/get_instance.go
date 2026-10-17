package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// Instance is the "Instance" use case.
type Instance struct {
	RegistryDeps
}

// Serve runs it.
func (uc Instance) Run(ctx context.Context, id string) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := uc.Connections.GetByID(ctx, id)
	if err != nil {
		return entity.Connection{}, err
	}
	mod, ok := uc.Modules.Serves(c.Driver)
	if !ok {
		return entity.Connection{}, constants.ErrUnsupported
	}
	readable, err := uc.CanRead(ctx, c.Name)
	if err != nil {
		return entity.Connection{}, err
	}
	if !readable {
		return entity.Connection{}, constants.ErrNotFound
	}
	c.Family = mod.Family()
	return c.Redacted(), nil
}
