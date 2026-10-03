package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The connection handlers are family-agnostic: the module decides which
// drivers belong to it, so the SQL and Redis namespaces share one
// implementation instead of two drifting copies.
func (d Deps) connections(ctx context.Context, mod interfaces.Module) (out []entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Connection, 0, len(list))
	for _, c := range list {
		if mod.Supports(c.Driver) {
			out = append(out, c.Redacted())
		}
	}
	return out, nil
}

func (d Deps) connection(ctx context.Context, mod interfaces.Module, name string) (out entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, mod, name)
	if err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}

// raw returns the password-bearing connection for a module: a connection of
// another engine family is not addressable in this namespace.
func (d Deps) raw(ctx context.Context, mod interfaces.Module, name string) (*entity.Connection, error) {
	c, err := d.Connections.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /v1/%s)",
			c.Name, mod.Family(), c.Driver, constants.FamilyOf(c.Driver))
	}
	return c, nil
}

// saveConnection upserts; an update that omits the password keeps the stored
// one (the console never sees it).
func (d Deps) saveConnection(ctx context.Context, mod interfaces.Module, c entity.Connection) (out entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if !mod.Supports(c.Driver) {
		return entity.Connection{}, fmt.Errorf("driver %q is not a %s driver (try /v1/%s)",
			c.Driver, mod.Family(), constants.FamilyOf(c.Driver))
	}
	if err := c.Validate(); err != nil {
		return entity.Connection{}, err
	}
	existing, err := d.Connections.Get(ctx, c.Name)
	switch {
	case err == nil:
		if c.Password == "" {
			c.Password = existing.Password
		}
		c.CreatedAt = existing.CreatedAt
	case errors.Is(err, interfaces.ErrNotFound):
		c.CreatedAt = time.Now().Unix()
	default:
		return entity.Connection{}, err
	}
	if err := d.Connections.Save(ctx, c); err != nil {
		return entity.Connection{}, err
	}
	return c.Redacted(), nil
}

// deleteConnection removes a connection and its changes after checking the
// connection belongs to this family, so the SQL namespace cannot delete a
// redis server by name.
func (d Deps) deleteConnection(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if _, err := d.raw(ctx, mod, name); err != nil {
		return err
	}
	return d.Connections.Delete(ctx, name)
}

func (d Deps) testConnection(ctx context.Context, mod interfaces.Module, name string) (err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, mod, name)
	if err != nil {
		return err
	}
	return mod.Ping(ctx, *c)
}

func (d Deps) schema(ctx context.Context, name string) (out []interfaces.Table, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, d.SQL, name)
	if err != nil {
		return nil, err
	}
	return d.SQL.Tables(ctx, *c)
}

func (d Deps) rows(ctx context.Context, name, schema, table string, limit int) (out interfaces.Result, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, d.SQL, name)
	if err != nil {
		return interfaces.Result{}, err
	}
	return d.SQL.Rows(ctx, *c, schema, table, limit)
}

// op dispatches one direct SQL operation.
func (d Deps) op(ctx context.Context, name, operation, sqlText, table string) (out interfaces.Result, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, d.SQL, name)
	if err != nil {
		return interfaces.Result{}, err
	}
	switch operation {
	case constants.OpQuery:
		return d.SQL.Query(ctx, *c, sqlText)
	case constants.OpAlter:
		return d.SQL.Alter(ctx, *c, sqlText)
	case constants.OpRollback:
		return d.SQL.DryRun(ctx, *c, sqlText)
	case constants.OpAnalytics:
		return d.SQL.Analyze(ctx, *c, table)
	}
	return interfaces.Result{}, fmt.Errorf("unknown op %q", operation)
}

// @Summary List connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/sql/connections [get]
func ListConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		conns, err := d.connections(ctx, d.SQL)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a connection
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections [post]
func SaveConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, d.SQL, c)
	})
}

// @Summary Get one connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/sql/connections/{name} [get]
func GetConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.connection(ctx, d.SQL, r.PathValue("name"))
	})
}

// @Summary Delete a connection (and its changes)
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name} [delete]
func DeleteConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		return d.deleteConnection(ctx, d.SQL, r.PathValue("name"))
	})
}

// @Summary Test that a connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/test [post]
func TestConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.testConnection(ctx, d.SQL, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary Introspect a connection's tables and columns
// @Param name path string true "connection name"
// @Success 200 {object} object{tables=[]sql.Table}
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/schema [get]
func GetSchema(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		tables, err := d.schema(ctx, r.PathValue("name"))
		return map[string]any{"tables": tables}, err
	})
}

// @Summary Preview a table's rows
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/tables/{table}/rows [get]
func GetRows(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return d.rows(ctx, r.PathValue("name"), r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	})
}

// @Summary Run a SQL console op (query | alter | rollback | analytics)
// @Param name path string true "connection name"
// @Param body body opBody true "sql (or table for analytics)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Apply a write directly (DML or DDL)
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/alter [post]
func PostAlter(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Dry-run statements inside a transaction that always rolls back
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/rollback [post]
func PostRollback(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Refresh planner statistics for a table
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/analytics [post]
func PostAnalytics(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// opRoutes pins each op route to its op name.
var opRoutes = map[string]string{
	"POST /v1/sql/connections/{name}/query":     constants.OpQuery,
	"POST /v1/sql/connections/{name}/alter":     constants.OpAlter,
	"POST /v1/sql/connections/{name}/rollback":  constants.OpRollback,
	"POST /v1/sql/connections/{name}/analytics": constants.OpAnalytics,
}

func doOp(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body opBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.op(ctx, r.PathValue("name"), opRoutes[r.Pattern], body.SQL, body.Table)
	})
}
