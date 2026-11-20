package web_test

import (
	"net/http"
	"net/http/httptest"
	"noxis/internal/web"
	"strings"
	"testing"
)

// The binary always has a page to serve: the placeholder is committed, and it names the API it was
// built for so an operator reading it knows where to point curl.
func TestPlaceholderIsServedAndSaysSo(t *testing.T) {
	console, err := web.New("http://localhost:9000")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if console.Built() {
		t.Fatal("the committed placeholder was reported as a real console")
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("content type = %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "not built") {
		t.Fatalf("the placeholder does not say it is one:\n%s", body)
	}
	if !strings.Contains(body, "http://localhost:9000") {
		t.Fatal("the placeholder does not name the API it was told to call")
	}
}

// A client-side route is the shell: the console decides what to render, so the server must not 404
// on the first refresh of a deep link.
func TestUnknownRouteFallsBackToTheShell(t *testing.T) {
	console, err := web.New("http://api")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, path := range []string{"/proposals", "/proposals/0192f0c0-0000-7000-8000-000000000001", "/nope"} {
		rec := httptest.NewRecorder()
		console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "not built") {
			t.Fatalf("%s answered %d, want the shell", path, rec.Code)
		}
	}
}

// The console is read-only, and it says so rather than pretending the write landed.
func TestWritesAreRefused(t *testing.T) {
	console, err := web.New("http://api")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		console.Handler().ServeHTTP(rec, httptest.NewRequest(method, "/", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s answered %d, want 405", method, rec.Code)
		}
		if rec.Header().Get("Allow") == "" {
			t.Fatalf("%s: no Allow header", method)
		}
	}
}

// A path that escapes the build is not served from outside it.
func TestPathTraversalIsNotServed(t *testing.T) {
	console, err := web.New("http://api")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/../../etc/passwd", nil))
	if strings.Contains(rec.Body.String(), "root:") {
		t.Fatal("the handler served a file from outside the build")
	}
}
