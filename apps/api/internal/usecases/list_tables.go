package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Schema is the "Schema" use case.
type Schema struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc Schema) Serve(ctx context.Context, sq interfaces.SQLModule, name string) (out []interfaces.Table, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := rawConn(ctx, uc.Connections, uc.Permissions, sq, name, entity.LevelRead)
	if err != nil {
		return nil, err
	}
	return sq.Tables(ctx, *c)
}
