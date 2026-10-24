package cache

import (
	"net/http"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Delete removes the selected keys only.
type Delete struct{ usecases.RegistryDeps }

// @Summary Remove keys (the selected ones only)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body entity.CacheDeleteRequest true "keys to remove"
// @Success 200 {object} entity.CacheDeleteResult
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.CacheDeleteRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	ctx = common.WithModuleFamily(ctx, r.PathValue("engine"))
	var mc interfaces.MemcacheModule
	if err := common.LoadModule(ctx, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	deleted, err := mc.Delete(ctx, *c, body.Keys)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.CacheDeleteResult{Deleted: deleted})
}
