package modules

import "net/http"

// registerNodes: POST /v1/nodes/{node}/register — one-shot node registration.
func registerNodes(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("POST /v1/nodes/{node}/register", func(w http.ResponseWriter, r *http.Request) {
		node := r.PathValue("node")
		if err := d.Repo.Register(r.Context(), node); err != nil {
			writeErr(w, err)
			return
		}
		d.Log.InfoContext(r.Context(), "node registered", "node", node)
		w.WriteHeader(http.StatusNoContent)
	})
}
