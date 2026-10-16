package middlewares

import (
	"encoding/json"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// WithAuth requires a valid bearer token on every public API request. There is
// no open mode: a nil verifier is a wiring bug and denies rather than trusting.
// The console shell, assets, /healthz and the two pre-token auth endpoints stay
// reachable so login can happen.
func WithAuth(verifier interfaces.Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, constants.APIPrefix+"/") || authPublic(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if verifier == nil {
			writeUnauthorized(w, "authentication is not configured")
			return
		}
		raw := bearerToken(r)
		if raw == "" {
			writeUnauthorized(w, "missing bearer token")
			return
		}
		principal, err := verifier.Verify(r.Context(), raw)
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(usecases.WithPrincipal(r.Context(), principal)))
	})
}

// authPublic reports whether a path under the API prefix is reachable without a
// token: the config probe and the login the console calls before it has one.
func authPublic(path string) bool {
	switch path {
	case constants.APIPrefix + "/auth/config", constants.APIPrefix + "/auth/login":
		return true
	}
	return false
}

// bearerToken returns the token from `Authorization: Bearer <token>`.
func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: message})
}
