package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

func TestGetMeServesPrincipal(t *testing.T) {
	deps, _, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Principal{
		Subject: "alice",
		Groups:  []string{"dev", "noxis-admins"},
		Admin:   true,
	}))
	rec := httptest.NewRecorder()
	depsHandler(mux, deps).ServeHTTP(rec, req.WithContext(usecases.WithDeps(req.Context(), deps)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"sub":"alice"`, `"admin":true`, `"noxis-admins"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("body %s missing %s", rec.Body.String(), want)
		}
	}
}

func TestGetMeIncludesLevels(t *testing.T) {
	deps, app, _, _ := newTestDeps(&fakeSQL{})
	app.Grants = &fakeGrants{grants: map[string]entity.Grant{
		"dev": {Connection: "demo", Group: "readers", Level: entity.LevelRead},
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Principal{Subject: "bob", Groups: []string{"readers"}}))
	rec := httptest.NewRecorder()
	depsHandler(mux, deps).ServeHTTP(rec, req.WithContext(usecases.WithDeps(req.Context(), deps)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"demo":"read"`) {
		t.Fatalf("levels missing: %s", rec.Body.String())
	}
}

func TestGetAuthConfig(t *testing.T) {
	deps, _, _, _ := newTestDeps(&fakeSQL{})
	deps.Auth = usecases.AuthInfo{Enabled: false}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/config", GetAuthConfig)

	rec := httptest.NewRecorder()
	depsHandler(mux, deps).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/config", http.NoBody))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMeUnauthenticated(t *testing.T) {
	deps, _, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", GetMe)

	rec := httptest.NewRecorder()
	depsHandler(mux, deps).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
