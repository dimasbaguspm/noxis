package search

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Targets lists what a search platform indexes, so a console can pick a target before it queries one.
type Targets struct{ usecases.RegistryDeps }

// @Summary List the indices a search platform holds
// @Param driver path string true "driver name (elasticsearch)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.SearchTargetList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/targets [get]
func (uc Targets) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var module interfaces.SearchEngine
	if err := uc.RouteModule(ctx, r, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	targets, err := module.Targets(ctx, *connection)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.SearchTargetList{Targets: targets})
}
