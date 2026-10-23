package middlewares

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

// discardLog is a logger whose output the test does not inspect.
func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

// WithAuth denies /external/v1/* when no verifier is wired: there is no open mode.
func TestWithAuthNilVerifierDenies(t *testing.T) {
	rec := httptest.NewRecorder()
	WithAuth(nil, okHandler(), discardLog()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/engine/sql/connections", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("nil verifier status=%d want=%d", rec.Code, http.StatusUnauthorized)
	}
}

// WithAuth guards /external/v1/* with a bearer token; the console shell, assets and
// health stay reachable so login can happen.
func TestWithAuthGuardsAPI(t *testing.T) {
	verifier := &fakeVerifier{err: errors.New("bad token")}
	h := WithAuth(verifier, okHandler(), discardLog())

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
	verifier := &fakeVerifier{principal: entity.Identity{Subject: "alice", Groups: []string{"dev"}, Role: entity.RoleSuper}}
	var got entity.Identity
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = usecases.Principal(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()
	WithAuth(verifier, inner, discardLog()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || got.Subject != "alice" || !got.IsSuper() {
		t.Fatalf("status=%d principal=%+v", rec.Code, got)
	}
}

// A refusal is logged where it happens, with a fixed reason and never the token.
func TestWithAuthRejectionIsLoggedWithoutTheToken(t *testing.T) {
	var buf bytes.Buffer
	verifier := &fakeVerifier{err: errors.New("bad token")}
	h := WithAuth(verifier, okHandler(), slog.New(slog.NewJSONHandler(&buf, nil)))

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer super-secret")
	h.ServeHTTP(httptest.NewRecorder(), req)

	line := buf.String()
	if !strings.Contains(line, "auth rejected") || !strings.Contains(line, "invalid or expired token") {
		t.Fatalf("log %s missing the rejection", line)
	}
	if strings.Contains(line, "super-secret") {
		t.Fatalf("log leaked the presented token: %s", line)
	}
}
