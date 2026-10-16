package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/common/config"
	"noxis/internal/entity"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

func TestGetMeServesPrincipal(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/me", GetMe)

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{
		Subject: "alice",
		Groups:  []string{"dev", "noxis-admins"},
		Admin:   true,
	}))
	rec := httptest.NewRecorder()
	depsHandler(mux, d).ServeHTTP(rec, req)

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
	base, _, _ := newTestBase(&fakeSQL{})
	base.Permissions = &fakePermissions{permissions: map[string]entity.Permission{
		"dev": {Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead},
	}}
	d := deps.Init(deps.Deps{Connections: base.Connections, Changes: base.Changes, Permissions: base.Permissions, Modules: modules.New(base.Modules, nil), Log: discardLog()})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/me", GetMe)

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{Subject: "bob", Groups: []string{"readers"}}))
	rec := httptest.NewRecorder()
	depsHandler(mux, d).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"demo":"read"`) {
		t.Fatalf("levels missing: %s", rec.Body.String())
	}
}

func TestGetAuthConfig(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	d.Auth = config.AuthInfo{Local: true}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/auth/config", GetAuthConfig)

	rec := httptest.NewRecorder()
	depsHandler(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/auth/config", http.NoBody))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"local":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMeUnauthenticated(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/me", GetMe)

	rec := httptest.NewRecorder()
	depsHandler(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
