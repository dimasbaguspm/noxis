package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"noxis/internal/application"
	"noxis/internal/entity"
)

type fakeVerifier struct {
	principal entity.Principal
	err       error
	calls     int
}

func (f *fakeVerifier) Verify(context.Context, string) (entity.Principal, error) {
	f.calls++
	if f.err != nil {
		return entity.Principal{}, f.err
	}
	return f.principal, nil
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

// TestWithAuthOpenMode: a nil verifier means no authentication.
func TestWithAuthOpenMode(t *testing.T) {
	rec := httptest.NewRecorder()
	withAuth(nil, okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/engine/sql/connections", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("open mode status=%d", rec.Code)
	}
}

// TestWithAuthGuardsAPI: the API needs a valid bearer token, everything else
// (console shell, health) does not.
func TestWithAuthGuardsAPI(t *testing.T) {
	verifier := &fakeVerifier{err: errors.New("bad token")}
	h := withAuth(verifier, okHandler())

	cases := []struct {
		name   string
		path   string
		header string
		want   int
	}{
		{"api missing token", "/v1/engine/sql/connections", "", http.StatusUnauthorized},
		{"api bad scheme", "/v1/engine/sql/connections", "Token abc", http.StatusUnauthorized},
		{"api invalid token", "/v1/engine/sql/connections", "Bearer nope", http.StatusUnauthorized},
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

// TestWithAuthAttachesPrincipal: a verified token reaches the handler as the
// request's principal.
func TestWithAuthAttachesPrincipal(t *testing.T) {
	verifier := &fakeVerifier{principal: entity.Principal{Subject: "alice", Groups: []string{"dev"}, Admin: true}}
	var got entity.Principal
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = application.Principal(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	withAuth(verifier, inner).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || got.Subject != "alice" || !got.Admin {
		t.Fatalf("status=%d principal=%+v", rec.Code, got)
	}
}
