package middlewares

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// WithAuth requires a valid bearer token on every public API request. There is
// no open mode: a nil verifier is a wiring bug and denies rather than trusting.
// The console shell, assets, /healthz and the two pre-token auth endpoints stay
// reachable so login can happen. A refusal is logged where it happens, because
// the request never reaches the router's request log.
func WithAuth(verifier interfaces.Verifier, next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, constants.APIPrefix+"/") || authPublic(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if verifier == nil {
			reject(w, r, log, "authentication is not configured")
			return
		}
		raw := bearerToken(r)
		if raw == "" {
			reject(w, r, log, "missing bearer token")
			return
		}
		principal, err := verifier.Verify(r.Context(), raw)
		if err != nil {
			reject(w, r, log, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(entity.WithPrincipal(r.Context(), principal)))
	})
}

// reject answers 401 and records why. The reason is a fixed phrase: the
// presented token is never logged.
func reject(w http.ResponseWriter, r *http.Request, log *slog.Logger, reason string) {
	log.WarnContext(r.Context(), "auth rejected", "path", r.URL.Path, "reason", reason)
	writeUnauthorized(w, reason)
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
