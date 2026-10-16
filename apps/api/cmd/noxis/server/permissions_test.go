package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/common/config"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

type fakePermissions struct {
	permissions map[string]entity.Permission
}

func (f *fakePermissions) ListPermissions(context.Context, string) ([]entity.Permission, error) {
	out := make([]entity.Permission, 0, len(f.permissions))
	for _, p := range f.permissions {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakePermissions) SetPermission(_ context.Context, p entity.Permission) error {
	f.permissions[p.Member] = p
	return nil
}

func (f *fakePermissions) DeletePermission(_ context.Context, _, member string) error {
	delete(f.permissions, member)
	return nil
}

func (f *fakePermissions) ListGroupNames(_ context.Context) ([]string, error) {
	seen := map[string]struct{}{}
	for _, p := range f.permissions {
		if p.MemberType == entity.MemberGroup {
			seen[p.Member] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for group := range seen {
		out = append(out, group)
	}
	return out, nil
}

func permissionRouter() http.Handler {
	d, _, _ := newTestDeps(&fakeSQL{})
	return depsHandler(newHandler(d, config.Config{}), d)
}

func request(t *testing.T, h http.Handler, method, path, body string, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{Subject: "alice", Admin: admin}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPermissionsRequireGlobalAdmin(t *testing.T) {
	h := permissionRouter()
	if rec := request(t, h, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", false); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin list status=%d want 403", rec.Code)
	}
	if rec := request(t, h, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"read"}`, false); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin set status=%d want 403", rec.Code)
	}
}

func TestPermissionsCRUDAsAdmin(t *testing.T) {
	h := permissionRouter()

	rec := request(t, h, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"write"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"member":"dev"`) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"root"}`, true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad level status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodDelete, "/external/v1/engine/sql/connections/demo/permissions?member=dev", "", true)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// A permission may target a single user by subject, not only a group.
func TestPermissionForUser(t *testing.T) {
	h := permissionRouter()

	rec := request(t, h, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"local:alice","memberType":"user","level":"admin"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("set user permission status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"memberType":"user"`) {
		t.Fatalf("user permission missing: %d %s", rec.Code, rec.Body.String())
	}
}
