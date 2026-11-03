package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// The operator console: a React+TS SPA (apps/web) built into bin/dist and
// synced into static/ by `make web`. Only placeholder.html, the icons and the
// logo are committed; the built shell and the bundles are gitignored.
//
//go:embed static
var consoleStatic embed.FS

// consoleRoot is the embedded console directory, resolved once: go:embed always
// carries static/, so an unusable root is a build defect rather than a request error.
var consoleRoot = func() fs.FS {
	sub, err := fs.Sub(consoleStatic, "static")
	if err != nil {
		panic("console static: " + err.Error())
	}
	return sub
}()

// consoleFiles serves the embedded directory without a listing.
var consoleFiles = http.FileServer(http.FS(consoleRoot))

// @Summary Liveness probe
// @Success 200 "process is alive"
// @Router /healthz [get]
func getHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// getConsole serves the operator console: an embedded file when the path names
// one (favicon, logo, hashed bundle), the SPA shell for a console route, and a
// JSON 404 for an unknown API path.
func getConsole(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		writeConsoleShell(w)
		return
	}
	if info, err := fs.Stat(consoleRoot, name); err == nil && !info.IsDir() {
		writeConsoleFile(w, r, name)
		return
	}
	if strings.HasPrefix(r.URL.Path, constants.APIPrefix+"/") {
		w.Header().Set(constants.HeaderContentType, constants.MediaTypeJSON)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: "no such endpoint: " + r.URL.Path})
		return
	}
	// A missing file is a 404, never the shell: a browser that asked for a
	// bundle or an icon must not be handed HTML with a 200.
	if strings.HasPrefix(name, constants.ConsoleAssets+"/") || path.Ext(name) != "" {
		http.NotFound(w, r)
		return
	}
	writeConsoleShell(w)
}

// writeConsoleFile serves one embedded console file. Only the content-hashed
// bundles are immutable; the icons and the shell keep a short lifetime, so a
// rebuilt console replaces them without a hard refresh.
func writeConsoleFile(w http.ResponseWriter, r *http.Request, name string) {
	if strings.HasPrefix(name, constants.ConsoleAssets+"/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	consoleFiles.ServeHTTP(w, r)
}

// writeConsoleShell serves the built SPA shell, or the placeholder when the
// console was never built into this binary.
func writeConsoleShell(w http.ResponseWriter) {
	for _, name := range []string{constants.ConsoleIndex, constants.ConsolePlaceholder} {
		data, err := fs.ReadFile(consoleRoot, name)
		if err != nil {
			continue
		}
		w.Header().Set(constants.HeaderContentType, constants.MediaTypeHTML)
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	http.Error(w, "console not built", http.StatusNotFound)
}
