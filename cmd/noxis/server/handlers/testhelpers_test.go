package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
)

// depsHandler wraps h so every request carries the test deps + registry,
// mirroring how the real server seeds the base request context in Run.
func depsHandler(h http.Handler, d deps.Uses) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := deps.Seed(r.Context(), d)
		h.ServeHTTP(w, r.WithContext(modules.WithRegistry(ctx, d.Modules)))
	})
}

// testCtx seeds the registry like Run does at boot, over the given families.
func testCtx(mods map[string]interfaces.Module) context.Context {
	return modules.WithRegistry(context.Background(), modules.New(mods))
}

// discardLog is the test logger the container carries.
func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
