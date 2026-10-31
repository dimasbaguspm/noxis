package mongo

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Databases lists databases with their sizes.
type Databases struct{ usecases.RegistryDeps }

// @Summary List databases with their sizes
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.DatabaseList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/databases [get]
func (uc Databases) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.DocumentBrowser](uc.RegistryDeps, w, r, "Databases", entity.LevelRead,
		func(ctx context.Context, module interfaces.DocumentBrowser, conn entity.Connection) (any, error) {
			databases, err := module.Databases(ctx, conn)
			return entity.DatabaseList{Databases: databases}, err
		})
}
