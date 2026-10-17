// Package schema holds the SQL schema and console-op use cases.
package schema

import (
	"fmt"
	"net/http"
	"strconv"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Tables introspects a connection's tables and columns.
type Tables struct{ usecases.RegistryDeps }

// @Summary Introspect a connection's tables and columns
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Success 200 {object} object{tables=[]entity.Table}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/schema [get]
func (uc Tables) Serve(w http.ResponseWriter, r *http.Request) {
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), sq, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	out, err := sq.Tables(r.Context(), *c)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, map[string]any{"tables": out})
}

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
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), sq, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := sq.Rows(r.Context(), *c, r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// opBody is the console op payload: sql for query/alter/rollback, table for
// analyze.
type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// opRoutes pins each op route to its operation name, read from the matched
// pattern (a request's Pattern is the route it hit).
var opRoutes = map[string]string{
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/query":    constants.OpQuery,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/alter":    constants.OpAlter,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/rollback": constants.OpRollback,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/analyze":  constants.OpAnalyze,
}

// Op runs a direct SQL console op.
type Op struct{ usecases.RegistryDeps }

// @Summary Run a SQL console op (query)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/query [post]
func (uc Op) Serve(w http.ResponseWriter, r *http.Request) {
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	var body opBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	operation := opRoutes[r.Pattern]
	c, err := uc.RawConn(r.Context(), sq, r.PathValue("name"), uc.OpLevel(operation))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	ctx := r.Context()
	switch operation {
	case constants.OpQuery:
		res, err := sq.Query(ctx, *c, body.SQL)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, res)
	case constants.OpAlter:
		res, err := sq.Alter(ctx, *c, body.SQL)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, res)
	case constants.OpRollback:
		res, err := sq.DryRun(ctx, *c, body.SQL)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, res)
	case constants.OpAnalyze:
		res, err := sq.Analyze(ctx, *c, body.Table)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, res)
	default:
		uc.Fail(w, fmt.Errorf("unknown op %q", operation))
	}
}

// OpAlter documents the alter route.
// @Summary Apply a write directly (DML or DDL)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/alter [post]
func OpAlter() {}

// OpRollback documents the rollback route.
// @Summary Dry-run statements inside a transaction that always rolls back
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/rollback [post]
func OpRollback() {}

// OpAnalyze documents the analyze route.
// @Summary Refresh planner statistics for a table
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/analyze [post]
func OpAnalyze() {}
