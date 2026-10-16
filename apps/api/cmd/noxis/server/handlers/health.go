package handlers

import "net/http"

// @Summary Liveness probe
// @Success 200 "process is alive"
// @Router /healthz [get]
func GetHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
