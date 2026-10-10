package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// SetGrant is the "SetGrant" use case.
type SetGrant struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc SetGrant) Serve(ctx context.Context, mod interfaces.Module, name, group, level string) (out entity.Grant, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return entity.Grant{}, err
	}
	if _, err := rawConn(ctx, uc.Connections, uc.Grants, mod, name, entity.LevelRead); err != nil {
		return entity.Grant{}, err
	}
	g := entity.Grant{Connection: name, Group: group, Level: level}
	if err := g.Validate(); err != nil {
		return entity.Grant{}, err
	}
	if err := uc.Grants.SetGrant(ctx, g); err != nil {
		return entity.Grant{}, err
	}
	return g, nil
}
