package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// TestConnection is the "TestConnection" use case.
type TestConnection struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc TestConnection) Serve(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := rawConn(ctx, uc.Connections, uc.Grants, mod, name, entity.LevelRead)
	if err != nil {
		return err
	}
	return mod.Ping(ctx, *c)
}
