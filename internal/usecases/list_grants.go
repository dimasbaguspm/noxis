package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ListGrants is the "ListGrants" use case.
type ListGrants struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc ListGrants) Serve(ctx context.Context, mod interfaces.Module, name string) (out []entity.Grant, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := Raw(ctx, uc.Connections, uc.Grants, mod, name, entity.LevelRead); err != nil {
		return nil, err
	}
	return uc.Grants.ListGrants(ctx, name)
}
