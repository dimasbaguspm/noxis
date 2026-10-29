package schema

import (
	"net/http"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Info reports what the connected server says about itself: the engine, the detected version and the capabilities that profile grants. Read-only, so a read grant is enough.
type Info struct{ usecases.RegistryDeps }

// @Summary What the connected server reports about itself
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.ServerInfo
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/info [get]
func (uc Info) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	ctx = common.WithModuleFamily(ctx, r.PathValue("engine"))
	var module interfaces.SQLModule
	if err := common.LoadModule(ctx, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	info, err := module.ServerInfo(ctx, *c)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, info)
}
