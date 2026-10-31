package cache

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Info reports what the cache server says about itself: memory, eviction, the hit ratio and the per-database key counts.
type Info struct{ usecases.RegistryDeps }

// @Summary Report what the cache server says about itself
// @Param driver path string true "driver name (redis, valkey, dragonfly)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.CacheInfo
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/stats [get]
func (uc Info) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var mc interfaces.KeyBrowser
	if err := uc.RouteModule(ctx, r, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := mc.Info(ctx, *c)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}
