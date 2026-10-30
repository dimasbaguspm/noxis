package cache

import (
	"fmt"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Flush empties the connected database (FLUSHDB, never FLUSHALL).
type Flush struct{ usecases.RegistryDeps }

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body entity.CacheFlushRequest true "confirm must repeat the connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/flush [post]
func (uc Flush) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.CacheFlushRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var mc interfaces.MemcacheModule
	if err := uc.RouteModule(ctx, r, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if body.Confirm != c.Name {
		uc.Fail(ctx, w, fmt.Errorf("flushing %s needs its name typed as confirmation", c.Name))
		return
	}
	if err := mc.Flush(ctx, *c); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "flushed"})
}
