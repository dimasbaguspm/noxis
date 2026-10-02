package modules

import (
	"encoding/json"
	"net/http"
)

// registerQuery: POST /v1/nodes/{node}/query runs SQL against an engine on
// the node, routed over its watch stream.
func registerQuery(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("POST /v1/nodes/{node}/query", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Engine string `json:"engine"`
			SQL    string `json:"sql"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, err)
			return
		}
		out, err := d.Ctrl.RouteQuery(r.Context(), r.PathValue("node"), body.Engine, body.SQL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"output": out})
	})
}
