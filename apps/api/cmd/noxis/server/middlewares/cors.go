// Package middlewares wraps the route table: CORS, request logging and bearer authentication.
package middlewares

import (
	"net/http"
	"strings"

	"noxis/internal/constants"
)

// CORS answers cross-origin browser calls (the console runs on another port) and short-circuits OPTIONS preflights.
func CORS(next http.Handler) http.Handler {
	allowMethods := strings.Join(constants.CORSMethods, ", ")
	allowHeaders := strings.Join(constants.CORSHeaders, ", ")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HeaderAccessControlAllowOrigin, "*")
		w.Header().Set(constants.HeaderAccessControlAllowMethods, allowMethods)
		w.Header().Set(constants.HeaderAccessControlAllowHeaders, allowHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
