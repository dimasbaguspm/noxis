package controller

import (
	"encoding/json"
	"net/http"

	"noxis/internal/entity"
	sqlops "noxis/internal/sql"
)

// @Summary List connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/connections [get]
func ListConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := From(r).Ctrl.Connections(r.Context())
	if err != nil {
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
	var conn entity.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeErr(w, err)
		return
	}
	saved, err := From(r).Ctrl.SaveConnection(r.Context(), conn)
	if err != nil {
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
	conn, err := From(r).Ctrl.Connection(r.Context(), r.PathValue("name"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

// @Summary Delete a connection
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name} [delete]
func DeleteConnection(w http.ResponseWriter, r *http.Request) {
	if err := From(r).Ctrl.DeleteConnection(r.Context(), r.PathValue("name")); err != nil {
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
	if err := From(r).Ctrl.TestConnection(r.Context(), r.PathValue("name")); err != nil {
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
	tables, err := From(r).Ctrl.Schema(r.Context(), r.PathValue("name"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// @Summary Run a doorman DB op (query | alter | rollback | analytics)
// @Param name path string true "connection name"
// @Param body body opBody true "sql (or table for analytics)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) { doOp(w, r) }

// @Summary Apply a write (DML or DDL)
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

// doOp pins the op by route and runs it through the SQL module.
func doOp(w http.ResponseWriter, r *http.Request) {
	op := map[string]string{
		"POST /v1/connections/{name}/query":     sqlops.OpQuery,
		"POST /v1/connections/{name}/alter":     sqlops.OpAlter,
		"POST /v1/connections/{name}/rollback":  sqlops.OpRollback,
		"POST /v1/connections/{name}/analytics": sqlops.OpAnalytics,
	}[r.Pattern]
	var body opBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, err)
		return
	}
	res, err := From(r).Ctrl.Op(r.Context(), r.PathValue("name"), op, body.SQL, body.Table)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
