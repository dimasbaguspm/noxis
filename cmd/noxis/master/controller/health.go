package controller

import (
	"context"
	"net/http"
	"time"
)

// @Summary Liveness probe
// @Success 200 "process is alive"
// @Router /healthz [get]
func GetHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// @Summary Readiness probe (sqlite ping)
// @Success 200 "store is reachable"
// @Failure 503 {object} map[string]string
// @Router /readyz [get]
func GetReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := From(r).Repo.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
}
