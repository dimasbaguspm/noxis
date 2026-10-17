package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// SQLChanges is the "SQLChanges" use case.
type SQLChanges struct {
	RegistryDeps
}

// Serve runs it.
func (uc SQLChanges) Run(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := uc.ReadableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if (all[i].Kind == constants.KindDML || all[i].Kind == constants.KindDDL) && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}
