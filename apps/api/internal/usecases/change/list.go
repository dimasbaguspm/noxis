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
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applying, applied, failed, rejected)"
// @Param kind query string false "dml or ddl (sql change engines only)"
// @Param q query string false "substring match on notes or statement"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} entity.ChangeList
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
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
	drivers, err := driverOf(ctx, uc.RegistryDeps)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	query := r.URL.Query()
	visible := readable(all, module, allowed, drivers, module.Driver())
	visible = filterByConnection(visible, query.Get("connection"))
	visible = filterByStatus(visible, query.Get("status"))
	kind, err := changeKind(query.Get("kind"), module)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	visible = filterByKind(visible, kind)
	visible = filterByText(visible, query.Get("q"))
	page, perPage := pagination.Params(query.Get("page"), query.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, entity.ChangeList{Changes: pagination.Slice(visible, page, perPage), Total: len(visible)})
}
