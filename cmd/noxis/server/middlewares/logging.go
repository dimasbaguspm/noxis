package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLog records method, path and duration of every request.
func RequestLog(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.InfoContext(r.Context(), "http",
			"method", r.Method, "path", r.URL.Path,
			"dur", time.Since(start).String())
	})
}
