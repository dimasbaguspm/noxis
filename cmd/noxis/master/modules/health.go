package modules

import (
	"context"
	"net/http"
	"time"
)

// health ports: liveness + readiness (sqlite ping).
func registerHealth(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := d.Repo.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
