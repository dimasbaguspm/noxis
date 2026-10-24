package mongo

import (
	"net/http"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Databases lists databases with their sizes.
type Databases struct{ usecases.RegistryDeps }

// @Summary List databases with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.DatabaseList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/databases [get]
func (uc Databases) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	ctx = common.WithModuleFamily(ctx, r.PathValue("engine"))
	var module interfaces.MongoModule
	if err := common.LoadModule(ctx, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	databases, err := module.Databases(ctx, *connection)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.DatabaseList{Databases: databases})
}
