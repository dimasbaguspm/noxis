package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// MongoChanges is the "MongoChanges" use case.
type MongoChanges struct{ Base }

// Serve runs it.
func (uc MongoChanges) Serve(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := uc.readableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == constants.KindMongo && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}
