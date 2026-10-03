// Package controller holds the REST handlers, one concern per file, with swag
// annotations (`make swagger` regenerates doc/swagger.yaml|json). Workflow
// logic lives next to the handlers; repositories and the SQL module arrive
// through the request context.
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

// Deps is everything a handler needs, injected per request: one module per
// engine family, plus the shared stores.
type Deps struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	SQL         interfaces.SQLModule
	Redis       interfaces.RedisModule
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
func From(r *http.Request) Deps { return r.Context().Value(ctxKey{}).(Deps) }

// module resolves the engine family that serves a driver, for the paths that
// work across families (connection reachability).
func (d Deps) module(driver string) interfaces.Module {
	if d.Redis.Supports(driver) {
		return d.Redis
	}
	return d.SQL
}

// span opens the span for the calling method (the name derives from the
// caller) and logs its outcome.
func (d Deps) span(ctx context.Context) (ctxOut context.Context, end func(error)) {
	return d.spanNamed(ctx, logging.CallerName(1))
}

func (d Deps) spanNamed(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, endSpan := logging.StartSpan(ctx, name)
	return ctxOut, func(err error) {
		if err != nil {
			d.Log.WarnContext(ctxOut, name, "error", err.Error())
		} else {
			d.Log.InfoContext(ctxOut, name)
		}
		endSpan(err)
	}
}

// handle runs one handler body: span (named after the handler), error
// mapping, JSON response.
func handle(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, d Deps) (any, error)) {
	d := From(r)
	ctx, end := d.spanNamed(r.Context(), logging.CallerName(1))
	v, err := fn(ctx, d)
	end(err)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// handleNoContent is handle for operations with an empty response.
func handleNoContent(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, d Deps) error) {
	d := From(r)
	ctx, end := d.spanNamed(r.Context(), logging.CallerName(1))
	err := fn(ctx, d)
	end(err)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

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
