// Package web is the console the binary carries. It serves whatever was built into internal/web/dist
// and says so out loud when nothing was: a 404 on the first thing an operator opens is a bug report
// about the wrong thing.
package web

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// assets is the console build. The placeholder is committed so the binary always has something to
// say; a real build replaces it and removes marker, which is how Built() knows.
//
//go:embed all:dist
var assets embed.FS

// Console answers the console's routes: a file when one exists, index.html when the path is a
// client-side route, and the placeholder page when there was no build at all.
type Console struct {
	dist  fs.FS
	index []byte
	built bool
}

// New reads the embedded build once. There is no configuration: the API serves the page, so the page
// already knows where its API is.
func New() (*Console, error) {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, err
	}
	return newFrom(dist)
}

// newFrom is New over any filesystem, so both branches — the committed placeholder and a real build —
// stay testable.
func newFrom(dist fs.FS) (*Console, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, errors.New("web: the embedded console has no index.html")
	}
	// NOT_BUILT is the only signal that separates a committed placeholder from a real build.
	_, markerErr := fs.Stat(dist, "NOT_BUILT")

	return &Console{dist: dist, index: index, built: markerErr != nil}, nil
}

// Built reports whether a real console was compiled in, which is what the serve command logs.
func (c *Console) Built() bool { return c.built }

// Handler answers files, then the single-page fallback, then nothing. `/config.json` is answered
// first and always: it is how a build learns which API it belongs to, and a console that was served
// from one machine and calls another cannot bake that into its bundle.
func (c *Console) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "the console is read-only", http.StatusMethodNotAllowed)
			return
		}

		if path.Clean(r.URL.Path) == "/config.json" {
			c.serveConfig(w)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		switch name {
		case "", "index.html":
			c.serveIndex(w)
			return
		}
		info, err := fs.Stat(c.dist, name)
		if err != nil || info.IsDir() {
			// A client-side route: the console decides what to render, so the shell is the answer.
			c.serveIndex(w)
			return
		}
		data, err := fs.ReadFile(c.dist, name)
		if err != nil {
			c.serveIndex(w)
			return
		}
		http.ServeContent(w, r, name, info.ModTime(), bytes.NewReader(data))
	})
}

// serveConfig is the deployment's answer to "which API am I?". It is not cached: a console that
// remembers the wrong server is a console that reports a network error nobody can explain.
//
// `api` is the origin the browser should call, and an empty one means "the origin you loaded this page
// from", which is the deployment this binary is. Nothing secret travels here: a bearer token in a GET
// response would be a token anybody who can load the page can copy.
func (*Console) serveConfig(w http.ResponseWriter) {
	body := []byte(`{"api":""}`)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (c *Console) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(c.index)
}
