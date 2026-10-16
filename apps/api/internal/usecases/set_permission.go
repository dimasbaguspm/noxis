package usecases

import (
	"context"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// SetPermission is the "SetPermission" use case.
type SetPermission struct {
	RegistryDeps
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve upserts one member's level on a connection.
func (uc SetPermission) Run(ctx context.Context, mod interfaces.Module, name, member, memberType, level string) (out entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return entity.Permission{}, err
	}
	if _, err := rawConn(ctx, uc.Connections, uc.Permissions, uc.Modules, mod, name, entity.LevelRead); err != nil {
		return entity.Permission{}, err
	}
	p := entity.Permission{Connection: name, Member: member, MemberType: memberType, Level: level}
	if err := p.Validate(); err != nil {
		return entity.Permission{}, err
	}
	if err := uc.Permissions.SetPermission(ctx, p); err != nil {
		return entity.Permission{}, err
	}
	return p, nil
}
