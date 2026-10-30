package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Get loads one change for the driver's queue.
type Get struct{ usecases.RegistryDeps }

// @Summary Get one change
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes/{id} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var change *entity.Change
	change, err = changeFor(ctx, uc.RegistryDeps, module, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, change)
}
