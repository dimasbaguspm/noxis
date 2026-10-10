package handlers

import (
	"context"
	"net/http"

	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

// depsHandler wraps h so every request carries the test deps, mirroring how
// the real server seeds the base request context in Run.
func depsHandler(h http.Handler, deps usecases.Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := usecases.WithDeps(r.Context(), deps)
		h.ServeHTTP(w, r.WithContext(modules.WithRegistry(ctx, deps.Engines)))
	})
}

// testCtx seeds the registry like Run does at boot, over the given families.
func testCtx(mods map[string]interfaces.Module) context.Context {
	return modules.WithRegistry(context.Background(), modules.New(mods))
}
