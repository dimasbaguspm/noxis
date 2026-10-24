package schema

import (
	"net/http"
	"strconv"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Rows previews a table's rows.
type Rows struct{ usecases.RegistryDeps }

// @Summary Preview a table's rows
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/tables/{table}/rows [get]
func (uc Rows) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	ctx = common.WithModuleFamily(ctx, r.PathValue("engine"))
	var module interfaces.SQLModule
	if err := common.LoadModule(ctx, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := module.Rows(ctx, *connection, r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, result)
}
