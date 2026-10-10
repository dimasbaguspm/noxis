package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// DeleteConnection is the "DeleteConnection" use case.
type DeleteConnection struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc DeleteConnection) Serve(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return err
	}
	if _, err := rawConn(ctx, uc.Connections, uc.Grants, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return uc.Connections.Delete(ctx, name)
}
