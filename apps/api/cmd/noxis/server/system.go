package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"noxis/internal/entity"
)

// The operator console: a React+TS SPA (web/) built into bin/dist and synced
// into static/ by `make web`. Only placeholder.html is committed (go:embed
// needs a file at compile time); the built SPA is gitignored.
//
//go:embed static
var consoleStatic embed.FS

// @Summary Liveness probe
// @Success 200 "process is alive"
// @Router /healthz [get]
func getHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// getConsole serves the built SPA shell, or the placeholder. Unknown API paths
// answer JSON 404 instead of the shell: a route that was renamed must fail
// loudly, not hand the caller an HTML page with 200.
func getConsole(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/external/v1/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: "no such endpoint: " + r.URL.Path})
		return
	}
	for _, name := range []string{"static/index.html", "static/placeholder.html"} {
		data, err := consoleStatic.ReadFile(name)
		if err != nil {
			continue
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	http.Error(w, "console not built", http.StatusNotFound)
}

// getAssets serves the built JS/CSS bundles under /assets/.
func getAssets(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(consoleStatic, "static")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(sub)).ServeHTTP(w, r)
}
