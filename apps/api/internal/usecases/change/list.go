package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists proposed changes for an engine (newest first).
type List struct{ usecases.RegistryDeps }

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applying, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} entity.ChangeList
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !supports(module) {
		uc.Fail(ctx, w, unsupportedChangeFlow(module))
		return
	}
	all, err := uc.Repositories.Changes.ListChanges(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	allowed, err := uc.ReadableConnections(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	query := r.URL.Query()
	visible := readable(all, module, allowed)
	visible = filterByConnection(visible, query.Get("connection"))
	visible = filterByStatus(visible, query.Get("status"))
	page, perPage := pagination.Params(query.Get("page"), query.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, entity.ChangeList{Changes: pagination.Slice(visible, page, perPage), Total: len(visible)})
}
