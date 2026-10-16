package usecases

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Raw is the connection gate the engine-scoped routes use: it resolves the
// connection, refuses a family mismatch, and enforces the level the operation
// needs, returning the password-bearing connection.
type Raw struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc Raw) Run(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	return rawConn(ctx, uc.Connections, uc.Permissions, mod, name, level)
}
