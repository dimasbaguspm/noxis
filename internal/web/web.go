// Package web is the console the binary carries. It serves whatever was built into internal/web/dist
// and says so out loud when nothing was: a 404 on the first thing an operator opens is a bug report
// about the wrong thing.
package web

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
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
	api   string
}

// New reads the embedded build once. apiURL is what the placeholder tells the reader to call.
func New(apiURL string) (*Console, error) {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, errors.New("web: the embedded console has no index.html")
	}
	// NOT_BUILT is the only signal that separates a committed placeholder from a real build.
	_, markerErr := fs.Stat(dist, "NOT_BUILT")

	page := index
	if markerErr == nil {
		// The placeholder names the API it was served for, so the page is useful on any host.
		rendered, err := template.New("index").Parse(string(index))
		if err != nil {
			return nil, err
		}
		var out strings.Builder
		if err := rendered.Execute(&out, map[string]string{"API": apiURL}); err != nil {
			return nil, err
		}
		page = []byte(out.String())
	}

	return &Console{dist: dist, index: page, built: markerErr != nil, api: apiURL}, nil
}

// Built reports whether a real console was compiled in, which is what the serve command logs.
func (c *Console) Built() bool { return c.built }

// API is the address the console was told to call.
func (c *Console) API() string { return c.api }

// Handler answers files, then the single-page fallback, then nothing.
func (c *Console) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "the console is read-only", http.StatusMethodNotAllowed)
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

func (c *Console) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(c.index)
}
