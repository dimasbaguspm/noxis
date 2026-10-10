package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Connection is the "Connection" use case.
type Connection struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc Connection) Serve(ctx context.Context, mod interfaces.Module, name string) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := Raw(ctx, uc.Connections, uc.Grants, mod, name, entity.LevelRead)
	if err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}
