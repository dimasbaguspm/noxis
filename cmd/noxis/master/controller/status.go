package controller

import "net/http"

// @Summary Node statuses (registry merged with live heartbeats)
// @Success 200 {object} object{nodes=[]entity.NodeStatus} "status of every node"
// @Failure 400 {object} map[string]string
// @Router /v1/status [get]
func GetStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := From(r).Ctrl.Statuses(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": rows})
}
