package schema

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Tables introspects a connection's tables and columns.
type Tables struct{ usecases.RegistryDeps }

// @Summary Introspect a connection's tables and columns
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.TableList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/schema [get]
func (uc Tables) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.SchemaReader](uc.RegistryDeps, w, r, "Tables", entity.LevelRead,
		func(ctx context.Context, module interfaces.SchemaReader, conn entity.Connection) (any, error) {
			tables, err := module.Tables(ctx, conn)
			return entity.TableList{Tables: tables}, err
		})
}
