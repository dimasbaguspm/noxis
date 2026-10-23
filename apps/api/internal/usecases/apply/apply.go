// Package apply serves manual change application.
package apply

import (
	"fmt"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Apply applies an approved change.
type Apply struct{ usecases.RegistryDeps }

// @Summary Apply an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/apply [post]
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
	flow, err := uc.flowFor(change)
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
func (uc Apply) flowFor(change *entity.Change) (interfaces.ChangeFlow, error) {
	flows := uc.Modules.Changes()
	if change.Kind == constants.KindMongo {
		if flows.Mongo == nil {
			return nil, fmt.Errorf("%w: no change queue for %s", constants.ErrUnsupported, change.Kind)
		}
		return flows.Mongo, nil
	}
	if flows.SQL == nil {
		return nil, fmt.Errorf("%w: no change queue for %s", constants.ErrUnsupported, change.Kind)
	}
	return flows.SQL, nil
}
