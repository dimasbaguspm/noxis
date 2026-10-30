// Package apply serves manual change application.
package apply

import (
	"context"
	"net/http"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Apply applies an approved change.
type Apply struct{ usecases.RegistryDeps }

// @Summary Apply an approved change
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes/{id}/apply [post]
func (uc Apply) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
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
	flow, err := flowFor(ctx, r.PathValue("driver"))
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

// flowFor is the change queue of the driver the route names, so the driver picks the queue rather than the change's kind.
func flowFor(ctx context.Context, driver string) (interfaces.ChangeFlow, error) {
	var flow interfaces.ChangeFlow
	if err := common.LoadModule(common.WithModuleDriver(ctx, driver), &flow); err != nil {
		return nil, err
	}
	return flow, nil
}
