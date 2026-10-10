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
	Grants      interfaces.GrantRepository
	Modules     map[string]interfaces.Module
}

// Serve runs it.
func (uc Schema) Serve(ctx context.Context, sq interfaces.SQLModule, name string) (out []interfaces.Table, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := Raw(ctx, uc.Connections, uc.Grants, uc.Modules, sq, name, entity.LevelRead)
	if err != nil {
		return nil, err
	}
	return sq.Tables(ctx, *c)
}
