package cache

import (
	"context"
	"net/http"
	"strconv"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Keys browses keys by pattern with a bounded scan.
type Keys struct{ usecases.RegistryDeps }

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Param cursor query int false "continuation from a previous page; 0 starts the scan"
// @Success 200 {object} entity.KeyPage
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/keys [get]
func (uc Keys) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "Keys", entity.LevelRead,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			cursor, _ := strconv.ParseUint(r.URL.Query().Get("cursor"), 10, 64)
			return mc.Keys(ctx, conn, r.URL.Query().Get("pattern"), cursor, limit)
		})
}
