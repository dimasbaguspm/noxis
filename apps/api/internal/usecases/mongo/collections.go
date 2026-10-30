package mongo

import (
	"net/http"

	"noxis/internal/common/logging"
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
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var module interfaces.DocumentBrowser
	if err := uc.RouteModule(ctx, r, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	collections, err := module.Collections(ctx, *connection, r.URL.Query().Get("database"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.CollectionList{Collections: collections})
}
