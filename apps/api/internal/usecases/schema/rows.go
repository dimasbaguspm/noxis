package schema

import (
	"context"
	"net/http"
	"strconv"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Rows previews a table's rows.
type Rows struct{ usecases.RegistryDeps }

// @Summary Preview a table's rows
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/tables/{table}/rows [get]
func (uc Rows) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.SchemaReader](uc.RegistryDeps, w, r, "Rows", entity.LevelRead,
		func(ctx context.Context, module interfaces.SchemaReader, conn entity.Connection) (any, error) {
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			return module.Rows(ctx, conn, r.URL.Query().Get("schema"), r.PathValue("table"), limit)
		})
}
