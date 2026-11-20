package middlewares_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"strings"
	"testing"
)

// The bearer token is the whole identity. These tests pin the three refusals apart, because "send a
// token" and "that token is not mine" send a reader to different people.

const opsSecret = "ops_secret_0123456789"

func newAuth(t *testing.T) *middlewares.Authenticator {
	t.Helper()
	auth, err := middlewares.NewAuthenticator([]config.Token{
		{Name: "ops", Secret: opsSecret, Scopes: []string{constants.ScopeRead, constants.ScopeAdmin}},
		{Name: "agent", Secret: "agent_secret_0123456789", Scopes: []string{constants.ScopeRead}},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	return auth
}

// refusal reads the envelope a refusal writes, so a test asserts the sentence a reader sees.
func refusal(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("the refusal is not the envelope: %v (%s)", err, rec.Body)
	}
	return envelope.Error.Message
}

func TestRequireSeparatesNoTokenFromAnUnknownOne(t *testing.T) {
	auth := newAuth(t)
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("a refused request reached the handler")
	})
	handler := auth.Require([]string{constants.ScopeRead}, next)

	for _, tc := range []struct {
		name    string
		header  string
		message string
	}{
		{"no header", "", "a bearer token is required"},
		{"not a bearer", "Basic abc", "a bearer token is required"},
		{"empty secret", "Bearer ", "a bearer token is required"},
		{"unknown secret", "Bearer guessed", "that token is not one this server accepts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
			if tc.header != "" {
				req.Header.Set(constants.HeaderAuthorization, tc.header)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if got := refusal(t, rec); got != tc.message {
				t.Fatalf("message = %q, want %q", got, tc.message)
			}
		})
	}
}

// A token the server issued, asking for a scope it does not hold, is a 403 that names the scope: the
// caller's token works, and what it lacks is a decision made elsewhere.
func TestRequireForbidsAScopeTheCallerLacks(t *testing.T) {
	auth := newAuth(t)
	handler := auth.Require([]string{constants.ScopeAdmin}, func(http.ResponseWriter, *http.Request) {
		t.Error("a forbidden request reached the handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer agent_secret_0123456789")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := refusal(t, rec); !strings.Contains(got, constants.ScopeAdmin) {
		t.Fatalf("message = %q, want it to name %q", got, constants.ScopeAdmin)
	}
}

// A token that holds every scope the route names hands the handler the caller, and the name it carries
// is the name that lands in the journal.
func TestRequireHandsTheCallerToTheHandler(t *testing.T) {
	auth := newAuth(t)
	var seen string
	handler := auth.Require([]string{constants.ScopeRead, constants.ScopeAdmin},
		func(w http.ResponseWriter, r *http.Request) {
			seen = middlewares.Caller(r.Context()).Name
			w.WriteHeader(http.StatusNoContent)
		})

	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	req.Header.Set(constants.HeaderAuthorization, "bearer "+opsSecret)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body)
	}
	if seen != "ops" {
		t.Fatalf("caller = %q, want ops", seen)
	}
}

// A public route reads the anonymous caller rather than a zero value nobody defined.
func TestCallerIsAnonymousWithoutAnAuthenticatedRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	if got := middlewares.Caller(req.Context()); got.HasAll([]string{constants.ScopeRead}) {
		if got := middlewares.Caller(req.Context()); got.HasAll([]string{constants.ScopeRead}) {
			t.Fatal("an anonymous caller holds a scope")
		}
	}
}

// A server with no declared token cannot authenticate anything, and says so before it listens.
func TestNewAuthenticatorRefusesAnEmptyEnvironment(t *testing.T) {
	if _, err := middlewares.NewAuthenticator(nil, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
		t.Fatal("an empty token list built an authenticator")
	}
}
