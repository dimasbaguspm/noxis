package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// CORS answers cross-origin preflights and stamps the allowed headers on every response.
func TestCORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	CORS(okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/external/v1/engine/postgres/connections", http.NoBody))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin=%q", got)
	}
}

func TestCORSHeadersOnRealRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	CORS(okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/engine/postgres/connections", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	for _, h := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
		if rec.Header().Get(h) == "" {
			t.Fatalf("missing header %s", h)
		}
	}
}
