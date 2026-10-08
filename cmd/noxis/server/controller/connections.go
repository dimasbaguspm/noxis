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

// The connection handlers are family-agnostic: the module named by the
// {engine} path parameter decides which drivers belong to it, so one
// implementation serves every family instead of a drifting copy per namespace.

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
		if other, ok := d.moduleByDriver(c.Driver); ok {
			return nil, fmt.Errorf("connection %q is not a %s connection (driver %q, try /v1/%s)",
				c.Name, mod.Family(), c.Driver, other.Family())
		}
		return nil, fmt.Errorf("connection %q is not a %s connection (driver %q)",
			c.Name, mod.Family(), c.Driver)
	}
	return c, nil
}

// saveConnection upserts; an update that omits the password keeps the stored
// one (the console never sees it).
func (d Deps) saveConnection(ctx context.Context, mod interfaces.Module, c entity.Connection) (out entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if !mod.Supports(c.Driver) {
		return entity.Connection{}, fmt.Errorf("driver %q is not a %s driver", c.Driver, mod.Family())
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
// connection belongs to this family, so one namespace cannot delete another
// family's server by name.
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

func (d Deps) schema(ctx context.Context, sq interfaces.SQLModule, name string) (out []interfaces.Table, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, sq, name)
	if err != nil {
		return nil, err
	}
	return sq.Tables(ctx, *c)
}

func (d Deps) rows(ctx context.Context, sq interfaces.SQLModule, name, schema, table string, limit int) (out interfaces.Result, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, sq, name)
	if err != nil {
		return interfaces.Result{}, err
	}
	return sq.Rows(ctx, *c, schema, table, limit)
}

// op dispatches one direct SQL operation.
func (d Deps) op(ctx context.Context, sq interfaces.SQLModule, name, operation, sqlText, table string) (out interfaces.Result, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, sq, name)
	if err != nil {
		return interfaces.Result{}, err
	}
	switch operation {
	case constants.OpQuery:
		return sq.Query(ctx, *c, sqlText)
	case constants.OpAlter:
		return sq.Alter(ctx, *c, sqlText)
	case constants.OpRollback:
		return sq.DryRun(ctx, *c, sqlText)
	case constants.OpAnalyze:
		return sq.Analyze(ctx, *c, table)
	}
	return interfaces.Result{}, fmt.Errorf("unknown op %q", operation)
}

// @Summary List connections (passwords redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/{engine}/connections [get]
func ListConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		conns, err := d.connections(ctx, mod)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/{engine}/connections [post]
func SaveConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, mod, c)
	})
}

// @Summary Get one connection (password redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name} [get]
func GetConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		return d.connection(ctx, mod, r.PathValue("name"))
	})
}

// @Summary Delete a connection (and its changes)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name} [delete]
func DeleteConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		mod, err := d.engine(r)
		if err != nil {
			return err
		}
		return d.deleteConnection(ctx, mod, r.PathValue("name"))
	})
}

// @Summary Test that a connection answers
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/test [post]
func TestConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		if err := d.testConnection(ctx, mod, r.PathValue("name")); err != nil {
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
// @Router /v1/{engine}/connections/{name}/schema [get]
func GetSchema(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		sq, err := d.engineSQL(r)
		if err != nil {
			return nil, err
		}
		tables, err := d.schema(ctx, sq, r.PathValue("name"))
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
// @Router /v1/{engine}/connections/{name}/tables/{table}/rows [get]
func GetRows(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		sq, err := d.engineSQL(r)
		if err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return d.rows(ctx, sq, r.PathValue("name"), r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	})
}

// @Summary Run a SQL console op (query | alter | rollback | analytics)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql (or table for analytics)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Apply a write directly (DML or DDL)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/alter [post]
func PostAlter(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Dry-run statements inside a transaction that always rolls back
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/rollback [post]
func PostRollback(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Refresh planner statistics for a table
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/analyze [post]
func PostAnalyze(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// opRoutes pins each op route to its op name.
var opRoutes = map[string]string{
	"POST /v1/{engine}/connections/{name}/query":    constants.OpQuery,
	"POST /v1/{engine}/connections/{name}/alter":    constants.OpAlter,
	"POST /v1/{engine}/connections/{name}/rollback": constants.OpRollback,
	"POST /v1/{engine}/connections/{name}/analyze":  constants.OpAnalyze,
}

func doOp(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		sq, err := d.engineSQL(r)
		if err != nil {
			return nil, err
		}
		var body opBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.op(ctx, sq, r.PathValue("name"), opRoutes[r.Pattern], body.SQL, body.Table)
	})
}
