package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// The placeholder only exists before a build, so a built console cannot exercise it. The branch still
// has to work: it is what an operator sees when they piped an image without running `make web`.
func TestPlaceholderRendersTheAPI(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": {Data: []byte(`<div id="root"></div><p>{{.API}}</p>`)},
		"NOT_BUILT":  {Data: []byte("")},
	}
	console, err := newFrom(dist, Config{API: "https://api.example.com"})
	if err != nil {
		t.Fatalf("newFrom: %v", err)
	}
	if console.Built() {
		t.Fatal("a placeholder was reported as a build")
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "https://api.example.com") {
		t.Fatalf("the placeholder does not name the API:\n%s", rec.Body.String())
	}
}

// A dist without an index is not a console, and the caller has to hear about it here rather than in a
// browser.
func TestAConsoleWithoutIndexIsAnError(t *testing.T) {
	if _, err := newFrom(fstest.MapFS{}, Config{API: "http://api"}); err == nil {
		t.Fatal("an empty dist built a console")
	}
}
