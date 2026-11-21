package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noxis/internal/web"
	"strings"
	"testing"
)

// The repository carries a placeholder, not a build: the image compiles the console (see the Dockerfile)
// and a clone that runs `go test` has never seen npm. `Built` is what tells the two apart.
func TestTheCommittedConsoleIsAPlaceholder(t *testing.T) {
	console, err := web.New(web.Config{API: "http://localhost:9000"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if console.Built() {
		t.Fatal("the committed placeholder was reported as a build")
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
	console, err := web.New(web.Config{API: "http://api"})
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
	console, err := web.New(web.Config{API: "http://api"})
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
	console, err := web.New(web.Config{API: "http://api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/../../etc/passwd", nil))
	if strings.Contains(rec.Body.String(), "root:") {
		t.Fatal("the handler served a file from outside the build")
	}
}

// A build cannot bake in the API it belongs to: the same binary is served from a laptop and from a
// VPS, and the browser needs the address it can reach. /config.json is that answer, uncached.
func TestConfigJSONNamesTheAPI(t *testing.T) {
	console, err := web.New(web.Config{API: "https://api.example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config.json", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("config.json = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", got)
	}
	if !strings.Contains(rec.Body.String(), `"api":"https://api.example.com"`) {
		t.Fatalf("body = %s", rec.Body)
	}
}

// Nothing secret is in this answer. The console used to be handed a CLI's token here when the request
// looked local; it now signs in with a pasted one, so the answer is the API prefix and nothing else.
func TestConfigJSONCarriesNoToken(t *testing.T) {
	console, err := web.New(web.Config{API: "/v1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// A request from the machine the binary runs on, which is the case that used to be trusted.
	request := httptest.NewRequest(http.MethodGet, "/config.json", nil)
	request.RemoteAddr = "127.0.0.1:5555"
	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, request)

	var answer map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatalf("config.json: %v", err)
	}
	if _, ok := answer["token"]; ok {
		t.Fatalf("config.json = %s, want no token field", rec.Body)
	}
	if answer["api"] != "/v1" {
		t.Fatalf("api = %q, want /v1", answer["api"])
	}
}
