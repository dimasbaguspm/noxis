package controller

import (
	"embed"
	"io/fs"
	"net/http"
)

// The operator console: a React+TS SPA (web/) built into bin/dist and synced
// into static/ by `make web`. Only placeholder.html is committed (go:embed
// needs a file at compile time); the built SPA is gitignored.
//
//go:embed static
var consoleStatic embed.FS

// GetConsole serves the built SPA shell, or the placeholder.
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
