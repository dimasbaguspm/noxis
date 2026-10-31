package schema

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// ERD returns a read-authorized diagram of declared keys and foreign keys without exposing the connection password.
type ERD struct{ usecases.RegistryDeps }

// @Summary One connection's schema graph
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.SchemaGraph
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/erd [get]
func (uc ERD) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.SchemaReader](uc.RegistryDeps, w, r, "ERD", entity.LevelRead,
		func(ctx context.Context, sq interfaces.SchemaReader, conn entity.Connection) (any, error) {
			tables, err := sq.Tables(ctx, conn)
			if err != nil {
				return nil, err
			}
			if tables == nil {
				tables = []entity.Table{}
			}
			assignTableIDs(tables)
			return entity.SchemaGraph{
				Connection: entity.SchemaConnection{
					Name: conn.Name, Driver: conn.Driver, Database: conn.Database,
				},
				Tables: tables,
			}, nil
		})
}
