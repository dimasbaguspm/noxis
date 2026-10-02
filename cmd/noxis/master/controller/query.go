package controller

import (
	"encoding/json"
	"net/http"
)

type queryRequest struct {
	Engine string `json:"engine"`
	SQL    string `json:"sql"`
}

type queryOutput struct {
	Output string `json:"output"`
}

// @Summary Run a query against an engine on the node (SQL engines)
// @Param node path string true "slave node id"
// @Param body body queryRequest true "engine and SQL"
// @Success 200 {object} queryOutput
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string "node not connected"
// @Router /v1/nodes/{node}/doorman/query [post]
func PostQuery(w http.ResponseWriter, r *http.Request) {
	var body queryRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, err)
		return
	}
	out, err := From(r).Ctrl.Query(r.Context(), r.PathValue("node"), body.Engine, body.SQL)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, queryOutput{Output: out})
}
