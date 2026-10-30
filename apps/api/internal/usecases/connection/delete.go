package connection

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Delete removes a connection and its changes.
type Delete struct{ usecases.RegistryDeps }

// @Summary Delete a connection (and its changes)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name} [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Repositories.Connections.Delete(ctx, r.PathValue("name")); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}
