package schema

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// ERD returns a read-authorized diagram of declared keys and foreign keys without exposing the connection password.
type ERD struct{ usecases.RegistryDeps }

// @Summary One connection's schema graph
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.SchemaGraph
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/erd [get]
func (uc ERD) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var sq interfaces.SchemaReader
	if err := uc.RouteModule(ctx, r, &sq); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, sq, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	tables, err := sq.Tables(ctx, *c)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if tables == nil {
		tables = []entity.Table{}
	}
	assignTableIDs(tables)
	uc.Respond(w, entity.SchemaGraph{
		Connection: entity.SchemaConnection{
			Name: c.Name, Driver: c.Driver, Database: c.Database,
		},
		Tables: tables,
	})
}
