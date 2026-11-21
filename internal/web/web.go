// Package web is the console the binary carries. It serves whatever was built into internal/web/dist
// and says so out loud when nothing was: a 404 on the first thing an operator opens is a bug report
// about the wrong thing.
package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
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
	cfg   Config
}

// Config is what the console is told about the deployment: the API the browser can reach, and — only
// when this process was started from a signed-in CLI — the token that session already holds.
type Config struct {
	API   string
	Token string
}

// New reads the embedded build once. cfg.API is what the placeholder tells the reader to call.
func New(cfg Config) (*Console, error) {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, err
	}
	return newFrom(dist, cfg)
}

// newFrom is New over any filesystem, so the placeholder branch — the one that renders an API address
// into a page — stays testable after a real build replaces the committed placeholder.
func newFrom(dist fs.FS, cfg Config) (*Console, error) {
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
		if err := rendered.Execute(&out, map[string]string{"API": cfg.API}); err != nil {
			return nil, err
		}
		page = []byte(out.String())
	}

	return &Console{dist: dist, index: page, built: markerErr != nil, cfg: cfg}, nil
}

// ParseAPI reads the API's public URL. It must be absolute and http or https, because the browser
// resolves it from the page's own origin: a bare host would be read as a path, and a trailing slash
// would produce a double slash on every call.
func ParseAPI(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("web: the console needs the API's public URL: pass --api https://api.example.com")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("web: --api %q is not a URL: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("web: --api %q is not http or https", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("web: --api %q has no host", raw)
	}
	return parsed.Scheme + "://" + parsed.Host + strings.TrimSuffix(parsed.Path, "/"), nil
}

// Built reports whether a real console was compiled in, which is what the serve command logs.
func (c *Console) Built() bool { return c.built }

// API is the address the console was told to call.
func (c *Console) API() string { return c.cfg.API }

// SignedIn reports whether this process was handed a token, which is what the serve command logs.
func (c *Console) SignedIn() bool { return c.cfg.Token != "" }

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
			c.serveConfig(w, r)
			return
		}

		// The console is a face of the same binary, and an operator's `HEALTHCHECK` cannot know which
		// face a container runs: this route is the one the image's probe falls back to. It says nothing
		// about the API — that is the API's own /healthz — only that this page is being served.
		if path.Clean(r.URL.Path) == "/healthz" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
			// The same envelope the API answers with, so one probe reads both faces: `Live` unwraps `data`.
			_, _ = w.Write([]byte(`{"data":{"status":"ok"}}`))
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
// The token travels here only for a request that came from the machine the binary runs on: a console
// on a VPS serves whoever can reach it, and a bearer token in a GET response is a token anybody can
// copy. From anywhere else the console asks for one, exactly as it did before.
func (c *Console) serveConfig(w http.ResponseWriter, r *http.Request) {
	answer := map[string]string{"api": c.cfg.API}
	if c.SignedIn() && loopback(r) {
		answer["token"] = c.cfg.Token
	}
	body, err := json.Marshal(answer)
	if err != nil {
		http.Error(w, "the console has no API configured", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// loopback reports whether the request came from this machine: the token is the CLI's own, and this
// is the one place it is handed to a browser.
func loopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (c *Console) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(c.index)
}
