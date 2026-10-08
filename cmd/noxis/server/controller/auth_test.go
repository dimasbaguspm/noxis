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

// TestGetMeIncludesLevels: a non-admin's readable connections and their
// effective levels ride along, so the console can gate actions.
func TestGetMeIncludesLevels(t *testing.T) {
	d, _, _ := newDeps(&fakeSQL{})
	d.Grants = &fakeGrants{grants: map[string]entity.Grant{
		"dev": {Connection: "demo", Group: "readers", Level: entity.LevelRead},
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(WithPrincipal(req.Context(), entity.Principal{Subject: "bob", Groups: []string{"readers"}}))
	rec := httptest.NewRecorder()
	WithDeps(mux, d).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"demo":"read"`) {
		t.Fatalf("levels missing: %s", rec.Body.String())
	}
}

// TestGetAuthConfig: the console reads the issuer and client id here; open mode
// reports disabled.
func TestGetAuthConfig(t *testing.T) {
	d, _, _ := newDeps(&fakeSQL{})
	d.Auth = AuthInfo{Enabled: false}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/config", GetAuthConfig)

	rec := httptest.NewRecorder()
	WithDeps(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/config", http.NoBody))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
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
