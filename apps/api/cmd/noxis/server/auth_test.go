package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/common/config"
	"noxis/internal/entity"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

func TestGetMeServesPrincipal(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	h := newHandler(d, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{
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

func TestGetMeIncludesLevels(t *testing.T) {
	base, _, _ := newTestBase(&fakeSQL{})
	base.Permissions = &fakePermissions{permissions: map[string]entity.Permission{
		"dev": {Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead},
	}}
	d := newApp(sources{Connections: base.Connections, Changes: base.Changes, Permissions: base.Permissions, Modules: modules.New(base.Modules, nil)})
	h := newHandler(d, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{Subject: "bob", Groups: []string{"readers"}}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"demo":"read"`) {
		t.Fatalf("levels missing: %s", rec.Body.String())
	}
}

func TestGetAuthConfig(t *testing.T) {
	base, _, _ := newTestBase(&fakeSQL{})
	d := newApp(sources{
		Connections: base.Connections, Changes: base.Changes, Permissions: base.Permissions,
		Modules: modules.New(base.Modules, nil),
		Auth:    config.AuthInfo{Local: true},
	})
	h := newHandler(d, config.Config{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/auth/config", http.NoBody))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"local":true`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMeUnauthenticated(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	h := newHandler(d, config.Config{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
