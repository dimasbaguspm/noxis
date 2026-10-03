package server

import (
	"log/slog"
	"net/http"
	"time"
)

// withMiddleware adds CORS and request logging.
func withMiddleware(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
		log.InfoContext(r.Context(), "http",
			"method", r.Method, "path", r.URL.Path,
			"dur", time.Since(start).String())
	})
}
