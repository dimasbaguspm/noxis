package usecases

import (
	"context"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ListPermissions is the "ListPermissions" use case.
type ListPermissions struct {
	RegistryDeps
}

// Serve runs it.
func (uc ListPermissions) Run(ctx context.Context, mod interfaces.Module, name string) (out []entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := uc.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := uc.RawConn(ctx, mod, name, entity.LevelRead); err != nil {
		return nil, err
	}
	return uc.Permissions.ListPermissions(ctx, name)
}
