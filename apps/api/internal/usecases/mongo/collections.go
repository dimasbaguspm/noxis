package mongo

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Collections lists a database's collections with their sizes.
type Collections struct{ usecases.RegistryDeps }

// @Summary List a database's collections with their sizes
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} entity.CollectionList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/collections [get]
func (uc Collections) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.DocumentBrowser](uc.RegistryDeps, w, r, "Collections", entity.LevelRead,
		func(ctx context.Context, module interfaces.DocumentBrowser, conn entity.Connection) (any, error) {
			collections, err := module.Collections(ctx, conn, r.URL.Query().Get("database"))
			return entity.CollectionList{Collections: collections}, err
		})
}
