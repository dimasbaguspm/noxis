package controller

import (
	"net/http"

	"noxis/internal/entity"
)

// @Summary Node statuses (reported runtime merged with registered nodes)
// @Success 200 {object} object{nodes=[]entity.NodeStatus} "status of every node"
// @Failure 400 {object} map[string]string
// @Router /v1/status [get]
func GetStatus(w http.ResponseWriter, r *http.Request) {
	d := From(r)
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
}
