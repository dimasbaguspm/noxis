// Package api is the transport layer of the noxis REST API: thin HTTP
// handlers, swag annotations and the engine-family lookup. Workflows live in
// internal/usecases; a Server here only parses the request, calls the app
// and encodes the response.
//
// Routes are generic over the engine family: a path parameter `{engine}` names
// the family, and every handler resolves its module from the registry instead
// of naming a family in code. A family that does not serve an operation is
// refused as unsupported (404), never routed to the wrong module.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"noxis/internal/usecases"
	"noxis/internal/common/logging"
	"noxis/internal/interfaces"
)

// Server holds everything a handler needs, injected at wiring time: the
// application plus the transport-only pieces (public OIDC config and the tick
// endpoint key). Handlers are methods on it, so no dependency rides the
// request context.
type Server struct {
	App *usecases.App
	Log *slog.Logger
	// Auth is the public OIDC configuration the console reads to start login.
	Auth AuthInfo
	// TickKey guards POST /internal/tick; empty disables the endpoint.
	TickKey string
}

// spanNamed opens a span with an explicit name and logs its outcome; the
// handler helpers pass the caller's name.
func (s *Server) spanNamed(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, endSpan := logging.StartSpan(ctx, name)
	return ctxOut, func(err error) {
		if err != nil {
			s.Log.WarnContext(ctxOut, name, "error", err.Error())
		} else {
			s.Log.InfoContext(ctxOut, name)
		}
		endSpan(err)
	}
}

// handle runs one handler body: span (named after the handler), error
// mapping, JSON response.
func (s *Server) handle(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) (any, error)) {
	ctx, end := s.spanNamed(r.Context(), logging.CallerName(1))
	v, err := fn(ctx)
	end(err)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// handleNoContent is handle for operations with an empty response.
func (s *Server) handleNoContent(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) {
	ctx, end := s.spanNamed(r.Context(), logging.CallerName(1))
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

// ErrForbidden re-exports the application policy error so handlers can check
// it; writeErr already maps it to 403.
var ErrForbidden = usecases.ErrForbidden

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	// a missing row and a missing engine/sub-resource are both "not here"
	if errors.Is(err, interfaces.ErrNotFound) || errors.Is(err, interfaces.ErrUnsupported) {
		code = http.StatusNotFound
	}
	if errors.Is(err, errUnauthorized) {
		code = http.StatusUnauthorized
	}
	if errors.Is(err, ErrForbidden) {
		code = http.StatusForbidden
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}