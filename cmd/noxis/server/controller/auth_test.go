package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/entity"
)

// TestGetMeServesPrincipal: the console reads its identity and groups here.
func TestGetMeServesPrincipal(t *testing.T) {
	d, _, _ := newDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)
	h := WithDeps(mux, d)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(WithPrincipal(req.Context(), entity.Principal{
		Subject: "alice",
		Groups:  []string{"dev", "noxis-admins"},
		Admin:   true,
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"sub":"alice"`, `"admin":true`, `"noxis-admins"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("body %s missing %s", rec.Body.String(), want)
		}
	}
}

// TestGetMeUnauthenticated: no principal means 401, never an empty principal.
func TestGetMeUnauthenticated(t *testing.T) {
	d, _, _ := newDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)

	rec := httptest.NewRecorder()
	WithDeps(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
