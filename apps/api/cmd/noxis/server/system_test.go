package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/constants"
)

// serveConsole sends one request through the console handler.
func serveConsole(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	getConsole(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return rec
}

// The favicon and the logo are real files in the embedded tree: they must come
// back as themselves, not as the SPA shell with an HTML content type.
func TestConsoleServesEmbeddedFiles(t *testing.T) {
	cases := []struct {
		path        string
		contentType string
	}{
		{"/favicon.ico", "icon"},
		{"/logo.svg", "image/svg+xml"},
	}
	for _, tc := range cases {
		rec := serveConsole(t, tc.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d want=200", tc.path, rec.Code)
		}
		if got := rec.Header().Get(constants.HeaderContentType); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s content-type=%q want %q", tc.path, got, tc.contentType)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("%s served an empty body", tc.path)
		}
		if rec.Header().Get("Cache-Control") == "" {
			t.Fatalf("%s has no cache policy", tc.path)
		}
	}
}

// A console route that names no file gets the shell, so a deep link works after a reload.
func TestConsoleFallsBackToShell(t *testing.T) {
	rec := serveConsole(t, "/connections/demo")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200", rec.Code)
	}
	if got := rec.Header().Get(constants.HeaderContentType); !strings.Contains(got, "text/html") {
		t.Fatalf("content-type=%q want text/html", got)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\">") {
		t.Fatalf("body is not the console shell: %s", rec.Body.String())
	}
}

// A missing bundle is a 404: a browser that asked for JavaScript must never be handed HTML.
func TestConsoleMissingAssetIs404(t *testing.T) {
	if rec := serveConsole(t, "/assets/index-DOES-NOT-EXIST.js"); rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", rec.Code)
	}
}

// An unknown API path answers JSON 404 instead of the HTML shell.
func TestConsoleUnknownAPIIsJSON404(t *testing.T) {
	rec := serveConsole(t, constants.APIPrefix+"/no-such-endpoint")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", rec.Code)
	}
	if got := rec.Header().Get(constants.HeaderContentType); !strings.HasPrefix(got, constants.MediaTypeJSON) {
		t.Fatalf("content-type=%q want %q", got, constants.MediaTypeJSON)
	}
	if !strings.Contains(rec.Body.String(), "no such endpoint") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
