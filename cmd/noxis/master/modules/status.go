package modules

import (
	"net/http"

	"noxis/internal/entity"
)

// registerStatus: GET /v1/status merges reported runtime state with the
// registration registry so offline-but-registered nodes stay visible.
func registerStatus(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, r *http.Request) {
		rows, err := d.Ctrl.Statuses(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		byNode := map[string]entity.NodeStatus{}
		for _, s := range rows {
			byNode[s.Node] = s
		}
		ids, err := d.Repo.ListIDs(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		for _, id := range ids {
			if _, ok := byNode[id]; !ok {
				byNode[id] = entity.NodeStatus{Node: id, State: "offline"}
			}
		}
		statuses := make([]entity.NodeStatus, 0, len(byNode))
		for _, s := range byNode {
			statuses = append(statuses, s)
		}
		writeJSON(w, http.StatusOK, map[string]any{"nodes": statuses})
	})
}
