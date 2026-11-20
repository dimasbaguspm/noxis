package handlers

import (
	"net/http"
	"noxis/internal/app/read"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"noxis/internal/httpapi/respond"
)

// RunQuery runs one statement through the read guard. It is the only route that takes SQL, and it
// answers with the rows it read plus `truncated`, never with a fake cursor.
func (h *Handlers) RunQuery(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.RunQuery")
	defer end(nil)
	r = r.WithContext(ctx)

	var body struct {
		SQL   string `json:"sql"`
		Limit int    `json:"limit,omitempty"`
	}
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
