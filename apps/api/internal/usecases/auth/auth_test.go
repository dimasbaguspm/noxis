package auth

import (
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"noxis/internal/common/testkit"
	"noxis/internal/entity"
	"noxis/internal/mocks"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

func TestMe(t *testing.T) {
	conns := mocks.NewMockConnectionRepository(gomock.NewController(t))
	deps := usecases.RegistryDeps{Repositories: repositories.Registry{Connections: conns}}
	mux := testkit.Mux("GET", "/external/v1/me", Me{RegistryDeps: deps}.Serve)

	// a super admin holds admin on every connection without a grant lookup
	conns.EXPECT().List(gomock.Any()).Return(nil, nil)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/me", "", &entity.Identity{Subject: "alice", Groups: []string{"dev"}, Role: entity.RoleSuper})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sub":"alice"`) {
		t.Fatalf("me status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(mux, http.MethodGet, "/external/v1/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d want 401", rec.Code)
	}
}

func TestConfig(t *testing.T) {
	c := Config{Auth: entity.AuthConfig{Local: true}}
	rec := testkit.Do(testkit.Mux("GET", "/external/v1/auth/config", c.Serve), http.MethodGet, "/external/v1/auth/config", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"local":true`) {
		t.Fatalf("config status=%d body=%s", rec.Code, rec.Body.String())
	}
}
