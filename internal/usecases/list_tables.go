package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Schema is the "Schema" use case.
type Schema struct{ Base }

// Serve runs it.
func (uc Schema) Serve(ctx context.Context, sq interfaces.SQLModule, name string) (out []interfaces.Table, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := uc.Raw(ctx, sq, name, entity.LevelRead)
	if err != nil {
		return nil, err
	}
	return sq.Tables(ctx, *c)
}
