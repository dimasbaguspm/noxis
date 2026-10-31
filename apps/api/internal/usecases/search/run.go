// Package search holds the use cases for search platforms: what a platform indexes, and one query against a target.
package search

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Run answers one query against a target. A read needs read access; a write is refused here by the engine, so the route
// asks for write access up front rather than letting the reviewer find out later.
type Run struct{ usecases.RegistryDeps }

// @Summary Query one index on a search platform
// @Param driver path string true "driver name (elasticsearch)"
// @Param name path string true "connection name"
// @Param body body entity.SearchRequest true "op, target, and the platform's own query document"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/search [post]
func (uc Run) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var request interfaces.SearchRequest
	if err := uc.Decode(r, &request); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var module interfaces.SearchEngine
	if err := uc.RouteModule(ctx, r, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	level := entity.LevelRead
	if module.IsWrite(request.Op) {
		level = entity.LevelWrite
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), level)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	result, err := module.Search(ctx, *connection, request)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, result)
}
