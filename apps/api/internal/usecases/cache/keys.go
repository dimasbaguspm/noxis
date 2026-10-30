package cache

import (
	"net/http"
	"strconv"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Keys browses keys by pattern with a bounded scan.
type Keys struct{ usecases.RegistryDeps }

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} entity.KeyPage
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [get]
func (uc Keys) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var mc interfaces.MemcacheModule
	if err := uc.RouteModule(ctx, r, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := mc.Keys(ctx, *c, r.URL.Query().Get("pattern"), limit)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}
