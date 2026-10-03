package controller

import (
	"embed"
	"io/fs"
	"net/http"
)

// The operator console: a React+TS SPA (see web/) built into bin/dist and
// synced into static/ by `make web` / the Dockerfile. Only
// static/placeholder.html is committed — go:embed needs something at compile
// time, and the placeholder keeps every build path (bare `go build`, tests)
// working while making "console not built" explicit instead of serving a
// stale UI. Served at / (the shell) and /assets/ (bundles); the API stays
// under /v1 and never goes through the SPA.
//
//go:embed static
var consoleStatic embed.FS

// GetConsole serves the built SPA shell, or the placeholder when the console
// has not been built.
func GetConsole(w http.ResponseWriter, r *http.Request) {
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

// GetAssets serves the built JS/CSS bundles under /assets/.
func GetAssets(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(consoleStatic, "static")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(sub)).ServeHTTP(w, r)
}
