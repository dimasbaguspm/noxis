package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// redacted ensures passwords never leave the API.
func redact(c entity.Connection) entity.Connection { return c.Redacted() }

// connections lists the registry (passwords redacted).
func (d Deps) connections(ctx context.Context) ([]entity.Connection, error) {
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.Connection, 0, len(list))
	for _, c := range list {
		out = append(out, redact(c))
	}
	return out, nil
}

// connection returns one redacted connection.
func (d Deps) connection(ctx context.Context, name string) (entity.Connection, error) {
	c, err := d.Connections.Get(ctx, name)
	if err != nil {
		return entity.Connection{}, err
	}
	return redact(*c), nil
}

// raw returns the password-bearing connection for SQL work.
func (d Deps) raw(ctx context.Context, name string) (*entity.Connection, error) {
	return d.Connections.Get(ctx, name)
}

// saveConnection validates and upserts; an update that omits the password
// keeps the stored one (the console never sees it).
func (d Deps) saveConnection(ctx context.Context, c entity.Connection) (entity.Connection, error) {
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
	return redact(c), nil
}

// deleteConnection removes the connection and its changes (cascade).
func (d Deps) deleteConnection(ctx context.Context, name string) error {
	return d.Connections.Delete(ctx, name)
}

// testConnection verifies the stored connection answers.
func (d Deps) testConnection(ctx context.Context, name string) error {
	c, err := d.raw(ctx, name)
	if err != nil {
		return err
	}
	return d.SQL.Ping(ctx, *c)
}

// schema introspects a connection for the console's browser.
func (d Deps) schema(ctx context.Context, name string) ([]interfaces.Table, error) {
	c, err := d.raw(ctx, name)
	if err != nil {
		return nil, err
	}
	return d.SQL.Tables(ctx, *c)
}

// rows previews a table's rows.
func (d Deps) rows(ctx context.Context, name, schema, table string, limit int) (interfaces.Result, error) {
	c, err := d.raw(ctx, name)
	if err != nil {
		return interfaces.Result{}, err
	}
	return d.SQL.Rows(ctx, *c, schema, table, limit)
}

// op runs one direct SQL operation (query | alter | rollback | analytics).
func (d Deps) op(ctx context.Context, name, operation, sqlText, table string) (interfaces.Result, error) {
	c, err := d.raw(ctx, name)
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

// ---- handlers ----

// @Summary List connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/connections [get]
func ListConnections(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.list")
	defer end(nil)
	conns, err := d.connections(ctx)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": conns})
}

// @Summary Create or update a connection
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/connections [post]
func SaveConnection(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.save")
	defer end(nil)
	var c entity.Connection
	if err := decode(r, &c); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	saved, err := d.saveConnection(ctx, c)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// @Summary Get one connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/connections/{name} [get]
func GetConnection(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.get")
	defer end(nil)
	c, err := d.connection(ctx, r.PathValue("name"))
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// @Summary Delete a connection (and its changes)
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name} [delete]
func DeleteConnection(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.delete")
	defer end(nil)
	if err := d.deleteConnection(ctx, r.PathValue("name")); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Test that a connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/test [post]
func TestConnection(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.test")
	defer end(nil)
	if err := d.testConnection(ctx, r.PathValue("name")); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// @Summary Introspect a connection's tables and columns
// @Param name path string true "connection name"
// @Success 200 {object} object{tables=[]sql.Table}
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/schema [get]
func GetSchema(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.schema")
	defer end(nil)
	tables, err := d.schema(ctx, r.PathValue("name"))
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

// @Summary Preview a table's rows
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/tables/{table}/rows [get]
func GetRows(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.rows")
	defer end(nil)
	limit := atoi(r.URL.Query().Get("limit"))
	res, err := d.rows(ctx, r.PathValue("name"), r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// @Summary Run a SQL console op (query | alter | rollback | analytics)
// @Param name path string true "connection name"
// @Param body body opBody true "sql (or table for analytics)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Apply a write directly (DML or DDL)
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/alter [post]
func PostAlter(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Dry-run statements inside a transaction that always rolls back
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/rollback [post]
func PostRollback(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Refresh planner statistics for a table
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/analytics [post]
func PostAnalytics(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// doOp pins the op by route and runs it through the SQL module.
func doOp(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "connections.op")
	defer end(nil)
	op := map[string]string{
		"POST /v1/connections/{name}/query":     constants.OpQuery,
		"POST /v1/connections/{name}/alter":     constants.OpAlter,
		"POST /v1/connections/{name}/rollback":  constants.OpRollback,
		"POST /v1/connections/{name}/analytics": constants.OpAnalytics,
	}[r.Pattern]
	var body opBody
	if err := decode(r, &body); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	res, err := d.op(ctx, r.PathValue("name"), op, body.SQL, body.Table)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
