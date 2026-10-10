package middlewares

import (
	"encoding/json"
	"net/http"
	"strings"

	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// WithAuth requires a valid bearer token on the API when a verifier is
// configured. A nil verifier means authentication is disabled (open mode); the
// console shell, assets and /healthz stay reachable so login can happen.
func WithAuth(verifier interfaces.Verifier, next http.Handler) http.Handler {
	if verifier == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
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
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
