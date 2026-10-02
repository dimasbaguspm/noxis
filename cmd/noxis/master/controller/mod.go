// Package controller holds the REST handlers, one per method, annotated with
// swag comments (`make swagger` regenerates doc/swagger.yaml|json). Route
// wiring lives in cmd/noxis/master/routes.go.
package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"noxis/cmd/noxis/control"
)

// Deps is everything a handler needs from the master.
type Deps struct {
	Ctrl *control.Control
	Log  *slog.Logger
}

type ctxKey struct{}

// WithDeps carries Deps per request; handlers retrieve them via From.
func WithDeps(next http.Handler, d Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, d)))
	})
}

// From returns the request's Deps.
func From(r *http.Request) Deps {
	return r.Context().Value(ctxKey{}).(Deps)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}
