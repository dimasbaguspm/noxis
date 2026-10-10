package api

import (
	"context"
	"net/http"
	"strconv"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// The connection handlers are family-agnostic: the module named by the
// {engine} path parameter decides which drivers belong to it, so one
// implementation serves every family instead of a drifting copy per namespace.

// @Summary List connections (passwords redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/engine/{engine}/connections [get]
func (s *Server) ListConnections(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		conns, err := s.App.ConnectionList(ctx, mod)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/engine/{engine}/connections [post]
func (s *Server) SaveConnection(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return s.App.SaveConnection(ctx, mod, c)
	})
}

// @Summary Get one connection (password redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name} [get]
func (s *Server) GetConnection(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		return s.App.Connection(ctx, mod, r.PathValue("name"))
	})
}

// @Summary Delete a connection (and its changes)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name} [delete]
func (s *Server) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	s.handleNoContent(w, r, func(ctx context.Context) error {
		mod, err := s.engine(r)
		if err != nil {
			return err
		}
		return s.App.DeleteConnection(ctx, mod, r.PathValue("name"))
	})
}

// @Summary Test that a connection answers
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/test [post]
func (s *Server) TestConnection(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		if err := s.App.TestConnection(ctx, mod, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary Introspect a connection's tables and columns
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Success 200 {object} object{tables=[]sql.Table}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/schema [get]
func (s *Server) GetSchema(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		sq, err := s.engineSQL(r)
		if err != nil {
			return nil, err
		}
		tables, err := s.App.Schema(ctx, sq, r.PathValue("name"))
		return map[string]any{"tables": tables}, err
	})
}

// @Summary Preview a table's rows
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/tables/{table}/rows [get]
func (s *Server) GetRows(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		sq, err := s.engineSQL(r)
		if err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return s.App.Rows(ctx, sq, r.PathValue("name"), r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	})
}

// @Summary Run a SQL console op (query | alter | rollback | analytics)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql (or table for analytics)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/query [post]
func (s *Server) PostQuery(w http.ResponseWriter, r *http.Request) { s.doOp(w, r) }

// @Summary Apply a write directly (DML or DDL)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/alter [post]
func (s *Server) PostAlter(w http.ResponseWriter, r *http.Request) { s.doOp(w, r) }

// @Summary Dry-run statements inside a transaction that always rolls back
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/rollback [post]
func (s *Server) PostRollback(w http.ResponseWriter, r *http.Request) { s.doOp(w, r) }

// @Summary Refresh planner statistics for a table
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/analyze [post]
func (s *Server) PostAnalyze(w http.ResponseWriter, r *http.Request) { s.doOp(w, r) }

type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// opRoutes pins each op route to its op name.
var opRoutes = map[string]string{
	"POST /v1/engine/{engine}/connections/{name}/query":    constants.OpQuery,
	"POST /v1/engine/{engine}/connections/{name}/alter":    constants.OpAlter,
	"POST /v1/engine/{engine}/connections/{name}/rollback": constants.OpRollback,
	"POST /v1/engine/{engine}/connections/{name}/analyze":  constants.OpAnalyze,
}

func (s *Server) doOp(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		sq, err := s.engineSQL(r)
		if err != nil {
			return nil, err
		}
		var body opBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return s.App.Op(ctx, sq, r.PathValue("name"), opRoutes[r.Pattern], body.SQL, body.Table)
	})
}