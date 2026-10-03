package controller

import (
	"encoding/json"
	"net/http"

	control "noxis/internal"
	"noxis/internal/modules/doorman"
)

// @Summary Run a doorman DB op (query | alter | rollback | analytics)
// @Param node path string true "slave node id"
// @Param body body control.OpRequest true "op, engine, and sql/table"
// @Success 200 {object} object{output=string} "result output"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/doorman/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) {
	doOp(w, r)
}

// @Summary Apply an ALTER TABLE on the node's engine
// @Param node path string true "slave node id"
// @Param body body control.OpRequest true "engine and SQL"
// @Success 200 {object} object{output=string} "result output"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/doorman/alter [post]
func PostAlter(w http.ResponseWriter, r *http.Request) {
	doOp(w, r)
}

// @Summary Dry-run a change inside a transaction that always rolls back
// @Param node path string true "slave node id"
// @Param body body control.OpRequest true "engine and SQL"
// @Success 200 {object} object{output=string} "result output"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/doorman/rollback [post]
func PostRollback(w http.ResponseWriter, r *http.Request) {
	doOp(w, r)
}

// @Summary Analyze a table (ANALYZE / ANALYZE TABLE)
// @Param node path string true "slave node id"
// @Param body body control.OpRequest true "engine and table"
// @Success 200 {object} object{output=string} "result output"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/doorman/analytics [post]
func PostAnalytics(w http.ResponseWriter, r *http.Request) {
	doOp(w, r)
}

// doOp pins the op by route and dispatches the request to the node.
func doOp(w http.ResponseWriter, r *http.Request) {
	op := map[string]string{
		"POST /v1/nodes/{node}/doorman/query":     doorman.OpQuery,
		"POST /v1/nodes/{node}/doorman/alter":     doorman.OpAlter,
		"POST /v1/nodes/{node}/doorman/rollback":  doorman.OpRollback,
		"POST /v1/nodes/{node}/doorman/analytics": doorman.OpAnalytics,
	}[r.Pattern]
	var req control.OpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	req.Op = op
	out, err := From(r).Ctrl.Request(r.Context(), r.PathValue("node"), req)
	if err != nil {
		From(r).Log.WarnContext(r.Context(), "doOp error", "err", err.Error())
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": out})
}
