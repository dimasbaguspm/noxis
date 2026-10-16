package usecases

import (
	"context"
	"fmt"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// DeletePermission is the "DeletePermission" use case.
type DeletePermission struct {
	RegistryDeps
}

// Serve removes one member's permission from a connection.
func (uc DeletePermission) Run(ctx context.Context, mod interfaces.Module, name, member string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := RequireAdmin(ctx); err != nil {
		return err
	}
	if member == "" {
		return fmt.Errorf("member is required")
	}
	if _, err := rawConn(ctx, uc.Connections, uc.Permissions, uc.Modules, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return uc.Permissions.DeletePermission(ctx, name, member)
}
