// Package modules splits the REST API per feature. Each feature registers
// its routes on the shared mux; adding one is a new file, not a new switch.
// Handlers are plain (w, r) funcs annotated with swag comments — `make
// swagger` regenerates doc/swagger.yaml|json from them.
package modules

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"noxis/internal/repository"
	rpc "noxis/internal/services/grpc"
)

// Deps is everything a feature module needs from the master.
type Deps struct {
	Ctrl *rpc.Server
	Repo *repository.SQLite
	Log  *slog.Logger
}

type ctxKey struct{}

// WithDeps carries Deps per request; handlers retrieve them via From.
func WithDeps(next http.Handler, d Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, d)))
	})
}

// From returns the module deps for the request.
func From(r *http.Request) Deps {
	return r.Context().Value(ctxKey{}).(Deps)
}

// Register mounts every feature module.
func Register(mux *http.ServeMux, d Deps) {
	registerHealth(mux, d)
	registerStatus(mux, d)
	registerSpec(mux, d)
	registerQuery(mux, d)
	registerNodes(mux, d)
	registerDocs(mux, d)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if strings.Contains(err.Error(), "not connected") {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
