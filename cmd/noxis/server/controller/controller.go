// Package controller holds the REST handlers, one concern per file, with swag
// annotations (`make swagger` regenerates doc/swagger.yaml|json). Workflow
// logic lives next to the handlers; repositories and the engine-family registry
// arrive through the request context.
//
// Routes are generic over the engine family: a path parameter `{engine}` names
// the family, and every handler resolves its module from the registry instead
// of naming a family in code. A family that does not serve an operation is
// refused as unsupported (404), never routed to the wrong module.
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/interfaces"
)

// Deps is everything a handler needs, injected per request: the engine-family
// registry, plus the shared stores. The registry is keyed by family, so one
// generic route serves every engine and the family is read from the request's
// {engine} path parameter.
type Deps struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Modules     map[string]interfaces.Module
	Log         *slog.Logger
	// TickKey guards POST /internal/tick; empty disables the endpoint.
	TickKey string
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

// engine resolves the module named by the request's {engine} path parameter.
func (d Deps) engine(r *http.Request) (interfaces.Module, error) {
	name := r.PathValue("engine")
	if mod, ok := d.Modules[name]; ok {
		return mod, nil
	}
	return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
}

// moduleByDriver resolves the family that serves a driver through the registry,
// rather than a hardcoded driver-to-family switch.
func (d Deps) moduleByDriver(driver string) (interfaces.Module, bool) {
	for _, mod := range d.Modules {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}

// sqlOf, mongoOf and memcacheOf narrow a module to the capability a handler
// needs; a module that does not serve it is unsupported (404).
func sqlOf(mod interfaces.Module) (interfaces.SQLModule, error) {
	m, ok := mod.(interfaces.SQLModule)
	if !ok {
		return nil, unsupported(mod, "sql operations")
	}
	return m, nil
}

func mongoOf(mod interfaces.Module) (interfaces.MongoModule, error) {
	m, ok := mod.(interfaces.MongoModule)
	if !ok {
		return nil, unsupported(mod, "mongodb operations")
	}
	return m, nil
}

func memcacheOf(mod interfaces.Module) (interfaces.MemcacheModule, error) {
	m, ok := mod.(interfaces.MemcacheModule)
	if !ok {
		return nil, unsupported(mod, "memory-cache operations")
	}
	return m, nil
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: engine %q does not serve %s", interfaces.ErrUnsupported, mod.Family(), what)
}

// engineSQL, engineMongo and engineMemcache resolve the request's engine and
// narrow it to the capability the handler serves.
func (d Deps) engineSQL(r *http.Request) (interfaces.SQLModule, error) {
	mod, err := d.engine(r)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func (d Deps) engineMongo(r *http.Request) (interfaces.MongoModule, error) {
	mod, err := d.engine(r)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func (d Deps) engineMemcache(r *http.Request) (interfaces.MemcacheModule, error) {
	mod, err := d.engine(r)
	if err != nil {
		return nil, err
	}
	return memcacheOf(mod)
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
	// a missing row and a missing engine/sub-resource are both "not here"
	if errors.Is(err, interfaces.ErrNotFound) || errors.Is(err, interfaces.ErrUnsupported) {
		code = http.StatusNotFound
	}
	if errors.Is(err, errUnauthorized) {
		code = http.StatusUnauthorized
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
