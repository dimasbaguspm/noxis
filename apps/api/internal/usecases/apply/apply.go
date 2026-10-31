// Package apply serves manual change application.
package apply

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
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
	out, err := uc.ApplyChange(ctx, change, entity.Actor(ctx))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}
