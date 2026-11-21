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

// The API URL is resolved by the browser, so it must be absolute and it must not end in a slash: a
// bare host would be read as a path, and a trailing slash produces a double slash on every call.
func TestParseAPIRefusesWhatABrowserCannotUse(t *testing.T) {
	for _, raw := range []string{"", "api.example.com", "/api", "ftp://api.example.com"} {
		if _, err := web.ParseAPI(raw); err == nil {
			t.Fatalf("ParseAPI(%q) accepted it", raw)
		}
	}
	for raw, want := range map[string]string{
		"https://api.example.com":       "https://api.example.com",
		"https://api.example.com/":      "https://api.example.com",
		"http://192.168.1.10:9000":      "http://192.168.1.10:9000",
		" https://api.example.com/api/": "https://api.example.com/api",
	} {
		got, err := web.ParseAPI(raw)
		if err != nil {
			t.Fatalf("ParseAPI(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("ParseAPI(%q) = %q, want %q", raw, got, want)
		}
	}
}

// The token is the CLI's, and it reaches a browser only when that browser is on the machine the binary
// runs on: a console on a VPS serves whoever can reach it.
func TestTheTokenIsOnlyHandedToLoopback(t *testing.T) {
	console, err := web.New(web.Config{API: "http://api", Token: "tok"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, tc := range []struct {
		name      string
		remote    string
		forwarded bool
		wantToken bool
	}{
		{"loopback", "127.0.0.1:5555", false, true},
		{"loopback v6", "[::1]:5555", false, true},
		{"another machine", "203.0.113.7:5555", false, false},
		// A reverse proxy, Docker's userland proxy: they all connect from a loopback address, so a request
		// that arrives forwarded is not the browser on this machine however local it looks.
		{"forwarded by a proxy", "127.0.0.1:5555", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/config.json", nil)
			request.RemoteAddr = tc.remote
			if tc.forwarded {
				request.Header.Set("X-Forwarded-For", "203.0.113.7")
			}
			rec := httptest.NewRecorder()
			console.Handler().ServeHTTP(rec, request)

			var answer map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
				t.Fatalf("config.json: %v", err)
			}
			if answer["api"] != "http://api" {
				t.Fatalf("api = %q", answer["api"])
			}
			if _, ok := answer["token"]; ok != tc.wantToken {
				t.Fatalf("token present = %v, want %v", ok, tc.wantToken)
			}
		})
	}
}

// The console is the other face of the same image, and a container that serves it gets the same
// HEALTHCHECK: /healthz is what the probe falls back to, in the API's own envelope so one client reads
// both. It says nothing about the API — that is the API's route — only that this page is being served.
func TestTheConsoleAnswersHealth(t *testing.T) {
	console, err := web.New(web.Config{API: "http://api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content type = %q", got)
	}
	var answer struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatalf("healthz: %v", err)
	}
	if answer.Data.Status != "ok" {
		t.Fatalf("status = %q, want ok in the API's envelope", answer.Data.Status)
	}
}
