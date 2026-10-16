package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

type fakeVerifier struct {
	principal entity.Identity
	err       error
	calls     int
}

func (f *fakeVerifier) Verify(context.Context, string) (entity.Identity, error) {
	f.calls++
	if f.err != nil {
		return entity.Identity{}, f.err
	}
	return f.principal, nil
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

// WithAuth denies /external/v1/* when no verifier is wired: there is no open mode.
func TestWithAuthNilVerifierDenies(t *testing.T) {
	rec := httptest.NewRecorder()
	WithAuth(nil, okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/engine/sql/connections", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("nil verifier status=%d want=%d", rec.Code, http.StatusUnauthorized)
	}
}

// WithAuth guards /external/v1/* with a bearer token; the console shell, assets and
// health stay reachable so login can happen.
func TestWithAuthGuardsAPI(t *testing.T) {
	verifier := &fakeVerifier{err: errors.New("bad token")}
	h := WithAuth(verifier, okHandler())

	cases := []struct {
		name   string
		path   string
		header string
		want   int
	}{
		{"api missing token", "/external/v1/engine/sql/connections", "", http.StatusUnauthorized},
		{"api bad scheme", "/external/v1/engine/sql/connections", "Token abc", http.StatusUnauthorized},
		{"api invalid token", "/external/v1/engine/sql/connections", "Bearer nope", http.StatusUnauthorized},
		{"console open", "/", "", http.StatusOK},
		{"health open", "/healthz", "", http.StatusOK},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s: status=%d want=%d body=%s", tc.name, rec.Code, tc.want, rec.Body.String())
		}
	}
}

// A verified token reaches the handler as the request's principal.
func TestWithAuthAttachesPrincipal(t *testing.T) {
	verifier := &fakeVerifier{principal: entity.Identity{Subject: "alice", Groups: []string{"dev"}, Admin: true}}
	var got entity.Identity
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = usecases.Principal(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	WithAuth(verifier, inner).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || got.Subject != "alice" || !got.Admin {
		t.Fatalf("status=%d principal=%+v", rec.Code, got)
	}
}
