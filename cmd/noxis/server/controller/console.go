package controller

import (
	"embed"
	"io/fs"
	"net/http"
)

// The operator console: a React+TS SPA (see web/) built into bin/dist and
// synced into static/ by `make web` / the Dockerfile. static/index.html is
// a committed dependency-free fallback page, so a bare `go build` always
// embeds something usable. Served at / (the shell) and /assets/ (bundles);
// the API stays under /v1 and does not go through the SPA.
//
//go:embed static
var consoleStatic embed.FS

// GetConsole serves the SPA shell (index.html).
func GetConsole(w http.ResponseWriter, r *http.Request) {
	data, err := consoleStatic.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "console not built", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
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
