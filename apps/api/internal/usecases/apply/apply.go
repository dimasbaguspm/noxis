// Package apply serves manual change application.
package apply

import (
	"context"
	"net/http"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Apply applies an approved change.
type Apply struct{ usecases.RegistryDeps }

func (uc Apply) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	change, err := changeFor(ctx, uc.RegistryDeps, module, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Authorize(ctx, change.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	flow, err := flowFor(ctx, change)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := flow.Apply(ctx, change, entity.Actor(ctx))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}

// flowFor is the queue that owns a change's kind.
func flowFor(ctx context.Context, change *entity.Change) (interfaces.ChangeFlow, error) {
	if change.Kind == constants.KindMongo {
		var flow interfaces.MongoChangeFlow
		if err := common.LoadModule(ctx, &flow); err != nil {
			return nil, err
		}
		return flow, nil
	}
	var flow interfaces.SQLChangeFlow
	if err := common.LoadModule(ctx, &flow); err != nil {
		return nil, err
	}
	return flow, nil
}
