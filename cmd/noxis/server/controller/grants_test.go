package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/entity"
)

type fakeGrants struct {
	grants map[string]entity.Grant
}

func (f *fakeGrants) ListGrants(context.Context, string) ([]entity.Grant, error) {
	out := make([]entity.Grant, 0, len(f.grants))
	for _, g := range f.grants {
		out = append(out, g)
	}
	return out, nil
}

func (f *fakeGrants) SetGrant(_ context.Context, g entity.Grant) error {
	f.grants[g.Group] = g
	return nil
}

func (f *fakeGrants) DeleteGrant(_ context.Context, _, group string) error {
	delete(f.grants, group)
	return nil
}

func grantRouter() http.Handler {
	d, _, _ := newDeps(&fakeSQL{})
	d.Grants = &fakeGrants{grants: map[string]entity.Grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/grants", ListGrants)
	mux.HandleFunc("PUT /v1/engine/{engine}/connections/{name}/grants", SetGrant)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/grants", DeleteGrant)
	return WithDeps(mux, d)
}

func request(t *testing.T, h http.Handler, method, path, body string, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(WithPrincipal(req.Context(), entity.Principal{Subject: "alice", Admin: admin}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestGrantsRequireGlobalAdmin: managing policy is not a per-connection right.
func TestGrantsRequireGlobalAdmin(t *testing.T) {
	h := grantRouter()
	if rec := request(t, h, http.MethodGet, "/v1/engine/sql/connections/demo/grants", "", false); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin list status=%d want 403", rec.Code)
	}
	if rec := request(t, h, http.MethodPut, "/v1/engine/sql/connections/demo/grants", `{"group":"dev","level":"read"}`, false); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin set status=%d want 403", rec.Code)
	}
}

// TestGrantsCRUDAsAdmin: the happy path, plus an unknown level refused as 400.
func TestGrantsCRUDAsAdmin(t *testing.T) {
	h := grantRouter()

	rec := request(t, h, http.MethodPut, "/v1/engine/sql/connections/demo/grants", `{"group":"dev","level":"write"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodGet, "/v1/engine/sql/connections/demo/grants", "", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"dev"`) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodPut, "/v1/engine/sql/connections/demo/grants", `{"group":"dev","level":"root"}`, true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad level status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodDelete, "/v1/engine/sql/connections/demo/grants?group=dev", "", true)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}
