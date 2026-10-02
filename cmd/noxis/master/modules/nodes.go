package modules

import "net/http"

// registerNodes wires the node registration feature route.
func registerNodes(mux *http.ServeMux, _ Deps) {
	mux.HandleFunc("POST /v1/nodes/{node}/register", postRegister)
}

// @Summary Register a slave node (one-shot)
// @Param node path string true "slave node id"
// @Success 204 "registered"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/register [post]
func postRegister(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	node := r.PathValue("node")
	if err := d.Repo.Register(r.Context(), node); err != nil {
		writeErr(w, err)
		return
	}
	d.Log.InfoContext(r.Context(), "node registered", "node", node)
	w.WriteHeader(http.StatusNoContent)
}
