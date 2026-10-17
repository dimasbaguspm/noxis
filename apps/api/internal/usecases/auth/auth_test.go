package auth

import (
	"net/http"
	"strings"
	"testing"

	"noxis/internal/common/config"
	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
)

func TestMe(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	mux := ucstest.Mux("GET", "/external/v1/me", Me{RegistryDeps: f.Deps()}.Serve)

	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/me", "", &entity.Identity{Subject: "alice", Groups: []string{"dev"}, Super: true})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sub":"alice"`) {
		t.Fatalf("me status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/external/v1/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d want 401", rec.Code)
	}
}

func TestConfig(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	c := Config{RegistryDeps: f.Deps(), Auth: config.AuthInfo{Local: true}}
	rec := ucstest.Do(ucstest.Mux("GET", "/external/v1/auth/config", c.Serve), http.MethodGet, "/external/v1/auth/config", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"local":true`) {
		t.Fatalf("config status=%d body=%s", rec.Code, rec.Body.String())
	}
}
