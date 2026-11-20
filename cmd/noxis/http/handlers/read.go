package handlers

import (
	"net/http"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/app/read"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
)

// RunQuery runs one statement through the read guard. It is the only route that takes SQL, and it
// answers with the rows it read plus `truncated`, never with a fake cursor.
// @Summary Run a read
// @Description One SELECT, EXPLAIN or SHOW. A write, a locking read or a compound statement is refused before it reaches the engine.
// @Tags reading
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Param body body handlers.QueryRequest true "the statement and a row ceiling"
// @Success 200 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/targets/{ref}/query [post]
func (h *Handlers) RunQuery(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.RunQuery")
	defer end(nil)
	r = r.WithContext(ctx)

	var body QueryRequest
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	rows, err := h.Read.Query.Run(r.Context(), read.QueryInput{
		Ref:   r.PathValue("ref"),
		SQL:   body.SQL,
		Limit: body.Limit,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, rows)
}

// ListDatabases answers the schemas one target can see.
// @Summary List schemas
// @Tags reading
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref}/databases [get]
func (h *Handlers) ListDatabases(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ListDatabases")
	defer end(nil)
	r = r.WithContext(ctx)

	databases, err := h.Read.Databases.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	if databases == nil {
		databases = []string{}
	}
	respond.OK(w, databases)
}

// ListTables answers the tables in one schema. It is a list of the engine's own rows, so it is
// bounded by nothing but the schema: there is no cursor to invent for a name-keyed list.
// @Summary List tables in a schema
// @Tags reading
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Param db path string true "schema name"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref}/databases/{db}/tables [get]
func (h *Handlers) ListTables(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ListTables")
	defer end(nil)
	r = r.WithContext(ctx)

	tables, err := h.Read.Tables.Run(r.Context(), read.TablesInput{
		Ref:      r.PathValue("ref"),
		Database: r.PathValue("db"),
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	if tables == nil {
		tables = []domain.Table{}
	}
	respond.OK(w, tables)
}

// ShowTable answers one table's columns, indexes and the DDL the engine would use — the same text
// a rehearsal rebuilds from.
// @Summary Describe one table
// @Description Columns, indexes and the DDL the engine would use to create it — the statement a rehearsal rebuilds from.
// @Tags reading
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Param db path string true "schema name"
// @Param table path string true "table name"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref}/databases/{db}/tables/{table} [get]
func (h *Handlers) ShowTable(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ShowTable")
	defer end(nil)
	r = r.WithContext(ctx)

	detail, err := h.Read.Table.Run(r.Context(), read.TableInput{
		Ref:      r.PathValue("ref"),
		Database: r.PathValue("db"),
		Table:    r.PathValue("table"),
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, detail)
}
