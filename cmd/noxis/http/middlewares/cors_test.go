package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"noxis/cmd/noxis/http/middlewares"
	"testing"
)

const console = "https://console.example.com"

// The API is served for one console, and the browser is the judge: an origin it was not told about
// gets no header at all, which the browser reports as a CORS failure rather than as a request it
// made and lost.
func TestCORSAnswersOnlyTheConfiguredOrigin(t *testing.T) {
	handler := middlewares.CORS([]string{console})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tc := range []struct {
		origin string
		want   string
	}{
		{console, console},
		{"https://elsewhere.example.com", ""},
		{"", ""},
	} {
		req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.want {
			t.Fatalf("origin %q: allow-origin = %q, want %q", tc.origin, got, tc.want)
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Fatalf("origin %q: credentials were allowed", tc.origin)
		}
	}
}

// A preflight is answered without reaching a handler: the browser asks about the method and the
// headers, and the route it names may not accept either of them directly.
func TestCORSPreflightStopsAtTheMiddleware(t *testing.T) {
	reached := false
	handler := middlewares.CORS([]string{console})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/targets", nil)
	req.Header.Set("Origin", console)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if reached {
		t.Fatal("a preflight reached the handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("the preflight named no methods")
	}
}
