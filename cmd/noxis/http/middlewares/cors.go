package middlewares

import (
	"net/http"
	"noxis/internal/constants"
	"slices"
	"strings"
)

// CORS is how a deployment that serves the console from somewhere other than the API says so. The
// normal shape — the console at `/`, on the same origin it calls — answers no header at all, because a
// browser never asks: an empty allow-list means same-origin only. Only an allow-listed origin is
// echoed, and credentials are never allowed: a bearer token in a header is not a cookie.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := strings.Join([]string{
		constants.HeaderAuthorization,
		constants.HeaderContentType,
		constants.HeaderRequestID,
	}, ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The answer depends on the request's Origin whether or not one was allowed, so a cache
			// in front of this must never share one answer between two origins.
			w.Header().Add("Vary", "Origin")
			if origin := r.Header.Get("Origin"); origin != "" && slices.Contains(origins, origin) {
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
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
