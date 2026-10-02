package controller

import "net/http"

// @Summary Register a slave node (one-shot)
// @Param node path string true "slave node id"
// @Success 204 "registered"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/register [post]
func PostRegister(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	node := r.PathValue("node")
	if err := d.Ctrl.RegisterNode(r.Context(), node); err != nil {
		writeErr(w, err)
		return
	}
	d.Log.InfoContext(r.Context(), "node registered", "node", node)
	w.WriteHeader(http.StatusNoContent)
}
