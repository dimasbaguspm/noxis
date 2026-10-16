package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Connection is the "Connection" use case.
type Connection struct {
	RegistryDeps
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc Connection) Run(ctx context.Context, mod interfaces.Module, name string) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := rawConn(ctx, uc.Connections, uc.Permissions, uc.Modules, mod, name, entity.LevelRead)
	if err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}
