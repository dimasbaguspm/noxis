// Package handlers is the transport layer of the noxis REST API: thin HTTP
// handlers, swag annotations and shared response glue. An handler is a plain
// func(w, r); everything it needs comes from the request context via
// usecases.From (deps seeded once at boot, never per request). Workflows live
// in internal/usecases.
//
// Routes are generic over the engine family: a path parameter `{engine}` names
// the family, and every handler resolves its module from the application
// registry instead of naming a family in code. A family that does not serve an
// operation is refused as unsupported (404), never routed to the wrong module.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// handle runs one handler body: span (named after the handler; StartSpan
// derives the name from the caller), error mapping, JSON response.
func handle(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) (any, error)) {
	ctx, end := logging.StartSpan(r.Context(), logging.CallerName(1))
	v, err := fn(ctx)
	end(err)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// handleNoContent is handle for operations with an empty response.
func handleNoContent(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) {
	ctx, end := logging.StartSpan(r.Context(), logging.CallerName(1))
	err := fn(ctx)
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

// errUnauthorized maps to 401: the tick endpoint is key-guarded and /v1/me
// needs a caller.
var errUnauthorized = errors.New("invalid or missing api key")

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	// a missing row and a missing engine/sub-resource are both "not here"
	if errors.Is(err, interfaces.ErrNotFound) || errors.Is(err, interfaces.ErrUnsupported) {
		code = http.StatusNotFound
	}
	if errors.Is(err, errUnauthorized) {
		code = http.StatusUnauthorized
	}
	if errors.Is(err, usecases.ErrForbidden) {
		code = http.StatusForbidden
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
