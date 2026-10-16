package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// DeleteConnection is the "DeleteConnection" use case.
type DeleteConnection struct {
	RegistryDeps
}

// Serve runs it.
func (uc DeleteConnection) Run(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := uc.RequireAdmin(ctx); err != nil {
		return err
	}
	if _, err := uc.RawConn(ctx, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return uc.Connections.Delete(ctx, name)
}
