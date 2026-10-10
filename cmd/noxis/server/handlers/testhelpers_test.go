package handlers

import (
	"net/http"

	"noxis/internal/usecases"
)

// depsHandler wraps h so every request carries the test deps, mirroring how
// the real server seeds the base request context in Run.
func depsHandler(h http.Handler, deps usecases.Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(usecases.WithDeps(r.Context(), deps)))
	})
}
