// Package controller holds the REST handlers, one per method, annotated with
// swag comments (`make swagger` regenerates doc/swagger.yaml|json). Route
// wiring lives in cmd/noxis/master/routes.go.
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	control "noxis/internal"
	"noxis/internal/interfaces"
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
	code := http.StatusBadRequest
	if errors.Is(err, interfaces.ErrNotFound) {
		code = http.StatusNotFound
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
