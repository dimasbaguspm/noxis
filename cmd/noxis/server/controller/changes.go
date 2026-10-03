package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"noxis/internal/entity"
)

// @Summary List proposed changes (newest first)
// @Success 200 {object} object{changes=[]entity.Change}
// @Router /v1/changes [get]
func ListChanges(w http.ResponseWriter, r *http.Request) {
	changes, err := From(r).Ctrl.Changes(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changes": changes})
}

// @Summary Submit a change and validate query + rollback (dry-run, nothing persists)
// @Param body body entity.Change true "connection, notes, query, rollback"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes [post]
func SubmitChange(w http.ResponseWriter, r *http.Request) {
	var ch entity.Change
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeErr(w, err)
		return
	}
	saved, err := From(r).Ctrl.SubmitChange(r.Context(), ch)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// @Summary Get one change
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/changes/{id} [get]
func GetChange(w http.ResponseWriter, r *http.Request) {
	ch, err := From(r).Ctrl.Change(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Approve a validated change (self-approval allowed)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/approve [post]
func ApproveChange(w http.ResponseWriter, r *http.Request) {
	ch, err := From(r).Ctrl.ApproveChange(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Apply an approved change (transactional)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/apply [post]
func ApplyChange(w http.ResponseWriter, r *http.Request) {
	ch, err := From(r).Ctrl.ApplyChange(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Reject (discard) a change
// @Param id path string true "change id"
// @Success 204 "discarded"
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id} [delete]
func RejectChange(w http.ResponseWriter, r *http.Request) {
	if err := From(r).Ctrl.RejectChange(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Preview a table's rows
// @Param name path string true "connection name"
// @Param table path string true "table name (schema.table allowed)"
// @Param schema query string false "schema (when table is unqualified)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} sql.Result
// @Failure 400 {object} map[string]string
// @Router /v1/connections/{name}/tables/{table}/rows [get]
func GetRows(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	res, err := From(r).Ctrl.Rows(r.Context(), r.PathValue("name"), r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
