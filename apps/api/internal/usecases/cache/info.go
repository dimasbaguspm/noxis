package cache

import (
	"context"
	"net/http"

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
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "Info", entity.LevelRead,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			return mc.Info(ctx, conn)
		})
}
