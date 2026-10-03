// Package controller holds the REST handlers, one concern per file,
// annotated with swag comments (`make swagger` regenerates
// doc/swagger.yaml|json). The workflow logic lives here with the handlers:
// repositories and the SQL module arrive through the request context (Deps),
// so there is no separate core layer to keep in sync.
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/interfaces"
)

// Deps is everything a handler needs: the repositories (the app's state),
// the SQL module, and the logger. Injected per request.
type Deps struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	SQL         interfaces.SQLModule
	Log         *slog.Logger
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

// span opens a named span for one operation and logs its outcome, so every
// log line carries trace_id/span_id.
func (d Deps) span(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, end = logging.StartSpan(ctx, name)
	return ctxOut, func(err error) {
		if err != nil {
			d.Log.WarnContext(ctx, name, "error", err.Error())
		} else {
			d.Log.InfoContext(ctx, name)
		}
		end(err)
	}
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
