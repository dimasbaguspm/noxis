package middlewares

import (
	"net/http"
	"slices"
	"strings"

	"noxis/internal/constants"
)

// CORS lets the standalone console, which is served from its own origin, call the API.
// Only an allow-listed origin is echoed, and credentials are never allowed: a bearer
// token in a header is not a cookie, so a browser never needs them.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := strings.Join([]string{
		constants.HeaderAuthorization,
		constants.HeaderContentType,
		constants.HeaderRequestID,
	}, ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && slices.Contains(origins, origin) {
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Add("Vary", "Origin")
				header.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				header.Set("Access-Control-Allow-Headers", allowed)
				header.Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
