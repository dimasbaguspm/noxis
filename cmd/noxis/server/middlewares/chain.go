package middlewares

import (
	"log/slog"
	"net/http"
)

// WithMiddleware is the standard chain around the route table: CORS first so
// OPTIONS preflights answer without logging noise, then request logging.
func WithMiddleware(next http.Handler, log *slog.Logger) http.Handler {
	return RequestLog(CORS(next), log)
}
