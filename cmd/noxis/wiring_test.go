package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/config"
	"path/filepath"
	"testing"
	"time"
)

// token is the secret the tests hand the authenticator: 32 bytes, because a shorter one is a config
// error and the composition root would fail before it built anything.
const token = "0123456789abcdef0123456789abcdef"

func testConfig() config.Config {
	return config.Config{
		QueryTimeout: time.Second,
		ProbeTimeout: time.Second,
		SecretKey:    bytes.Repeat([]byte{7}, 32),
		Tokens: []config.Token{
			{Name: "ops", Secret: token, Scopes: []string{"read", "admin", "approve", "apply"}},
			{Name: "agent", Secret: token, Scopes: []string{"read", "propose"}},
		},
	}
}

// The composition root is the one place that names an adapter, so this is the one test that proves
// the whole chain still connects: config → sqlite → use cases → transport, answering a real route.
func TestBuildServerAnswersTheHealthRoute(t *testing.T) {
	handler, closer, err := buildServer(context.Background(), testConfig(),
		filepath.Join(t.TempDir(), "state.db"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}
	defer func() { _ = closer.Close() }()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d: %s", rec.Code, rec.Body)
	}
}

// A route behind a scope is refused without a token, through the real middleware rather than a
// handler that was handed a caller.
func TestBuildServerRefusesAnAnonymousCall(t *testing.T) {
	handler, closer, err := buildServer(context.Background(), testConfig(),
		filepath.Join(t.TempDir(), "state.db"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}
	defer func() { _ = closer.Close() }()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/targets", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /v1/targets = %d, want 401", rec.Code)
	}

	signed := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	signed.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, signed)
	if rec.Code != http.StatusOK {
		t.Fatalf("signed /v1/targets = %d: %s", rec.Code, rec.Body)
	}
}

// A state file that cannot be opened is the answer, not a handler serving half a product.
func TestBuildServerReportsABadStateFile(t *testing.T) {
	_, _, err := buildServer(context.Background(), testConfig(), t.TempDir(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("a directory as the state file built a server")
	}
}
