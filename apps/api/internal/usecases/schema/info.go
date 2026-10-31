package schema

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Info reports what the connected server says about itself: the engine, the detected version and the capabilities that profile grants. Read-only, so a read grant is enough.
type Info struct{ usecases.RegistryDeps }

// @Summary What the connected server reports about itself
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.ServerInfo
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/info [get]
func (uc Info) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.ServerReporter](uc.RegistryDeps, w, r, "Info", entity.LevelRead,
		func(ctx context.Context, module interfaces.ServerReporter, conn entity.Connection) (any, error) {
			return module.ServerInfo(ctx, conn)
		})
}
