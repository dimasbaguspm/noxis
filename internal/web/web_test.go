package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noxis/internal/web"
	"strings"
	"testing"
)

// A build is served from the binary, and the binary knows it is a build: `Built` is what the serve
// command logs, and what told an operator the placeholder was a placeholder.
func TestTheBuildIsServed(t *testing.T) {
	console, err := web.New(web.Config{API: "http://localhost:9000"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !console.Built() {
		t.Fatal("the committed console build was reported as a placeholder")
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
	if !strings.Contains(body, `id="root"`) {
		t.Fatalf("the build has no mount point:\n%s", body)
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
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="root"`) {
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
		wantToken bool
	}{
		{"loopback", "127.0.0.1:5555", true},
		{"loopback v6", "[::1]:5555", true},
		{"another machine", "203.0.113.7:5555", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/config.json", nil)
			request.RemoteAddr = tc.remote
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
