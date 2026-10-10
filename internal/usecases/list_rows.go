package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Rows is the "Rows" use case.
type Rows struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
	Modules     map[string]interfaces.Module
}

// Serve runs it.
func (uc Rows) Serve(ctx context.Context, sq interfaces.SQLModule, name, schema, table string, limit int) (out interfaces.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := Raw(ctx, uc.Connections, uc.Grants, uc.Modules, sq, name, entity.LevelRead)
	if err != nil {
		return interfaces.Result{}, err
	}
	return sq.Rows(ctx, *c, schema, table, limit)
}
