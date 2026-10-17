package usecases

import (
	"context"
	"errors"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
)

// RotatePassword is the "RotatePassword" use case.
type RotatePassword struct {
	RegistryDeps
	Connections interfaces.ConnectionRepository
}

// Serve runs it.
func (uc RotatePassword) Run(ctx context.Context, id, password string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return err
	}
	if password == "" {
		return errors.New("password is required")
	}
	c, err := uc.Connections.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, ok := uc.Modules.Serves(c.Driver); !ok {
		return constants.ErrUnsupported
	}
	c.Password = password
	return uc.Connections.Save(ctx, *c)
}
