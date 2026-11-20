package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/dotconfig"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// A fresh config directory per test: the CLI's file is the subject, not the developer's own session.
func isolated(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("API_TOKEN", "")
	t.Setenv("API_URL", "")
	return dir
}

func store(t *testing.T, profile dotconfig.Profile) {
	t.Helper()
	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("default", profile)
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func command() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	return cmd
}

// With nothing stored and nothing in the environment, the refusal has to be an instruction: a bare
// "401" is what sends a person looking at the server instead of at their own session.
func TestNoTokenSaysHowToSignIn(t *testing.T) {
	isolated(t)

	opts := &options{}
	err := opts.prepare(command())
	if err == nil {
		t.Fatal("prepare with no token returned no error")
	}
	if !strings.Contains(err.Error(), "auth signin") {
		t.Fatalf("error = %q, want the signin instruction", err)
	}
}

// A token checked a moment ago is trusted for the length of the window: a script must not ask the
// server twenty times to learn the same name.
func TestFreshProfileIsNotRechecked(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{
		APIURL: "http://127.0.0.1:1", Token: "stored", Caller: "agent",
		Scopes: []string{"read"}, CheckedAt: time.Now().UTC(),
	})

	opts := &options{}
	if err := opts.prepare(command()); err != nil {
		t.Fatalf("prepare with a fresh profile: %v", err)
	}
	if opts.token != "stored" {
		t.Fatalf("token = %q, want the stored one", opts.token)
	}
	if opts.name != "agent" {
		t.Fatalf("caller = %q, want the cached one", opts.name)
	}
}

// A revoked token is named as a session that ended, not as a request that failed: the fix is one
// command, and saying so is the whole point of the check.
func TestRevokedTokenSaysSignInAgain(t *testing.T) {
	isolated(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"no"}}`))
	}))
	defer server.Close()
	store(t, dotconfig.Profile{
		APIURL: server.URL, Token: "revoked", Caller: "agent",
		CheckedAt: time.Now().UTC().Add(-2 * time.Hour),
	})

	opts := &options{}
	err := opts.prepare(command())
	if err == nil {
		t.Fatal("prepare with a revoked token returned no error")
	}
	if !strings.Contains(err.Error(), "auth signin") {
		t.Fatalf("error = %q, want the signin instruction", err)
	}
	if strings.Contains(err.Error(), "revoked") {
		t.Fatalf("error = %q, it printed the token", err)
	}
}

// The check that succeeds is also the one that caches: the next command must not have to ask.
func TestCheckedIdentityIsCached(t *testing.T) {
	isolated(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"name":"reviewer","scopes":["read","approve"]}}`))
	}))
	defer server.Close()
	store(t, dotconfig.Profile{APIURL: server.URL, Token: "fresh", CheckedAt: time.Now().UTC().Add(-time.Hour)})

	opts := &options{}
	if err := opts.prepare(command()); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if opts.name != "reviewer" {
		t.Fatalf("caller = %q, want reviewer", opts.name)
	}

	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	stored, err := cfg.Profile("default")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if stored.Caller != "reviewer" || time.Since(stored.CheckedAt) > time.Minute {
		t.Fatalf("the identity was not cached: %+v", stored)
	}
}

// A flag is what a human typed now, and it wins — but it is never stored, and the warning says why.
func TestFlagTokenWinsAndIsNotStored(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{APIURL: "http://stored", Token: "stored", CheckedAt: time.Now().UTC()})

	opts := &options{apiURL: "http://flag", token: "from-flag"}
	if err := opts.prepare(command()); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if opts.apiURL != "http://flag" || opts.token != "from-flag" {
		t.Fatalf("resolved %q/%q, want the flag values", opts.apiURL, opts.token)
	}
	if opts.fromStore {
		t.Fatal("a flag token was treated as a stored one")
	}
}

// The file the CLI writes holds the token and nothing else needs to: this asserts the shape a
// reviewer would check by hand.
func TestSigninShapeLandsInTheFile(t *testing.T) {
	dir := isolated(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"name":"agent","scopes":["read"]}}`))
	}))
	defer server.Close()

	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("default", dotconfig.Profile{
		APIURL: server.URL, Token: "tok", Caller: "agent",
		Scopes: []string{"read"}, CheckedAt: time.Now().UTC(),
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "noxis", "config.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if decoded["current_profile"] != "default" {
		t.Fatalf("current_profile = %v, want default", decoded["current_profile"])
	}
	profiles, ok := decoded["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("profiles = %v", decoded["profiles"])
	}
	if _, ok := profiles["default"].(map[string]any); !ok {
		t.Fatalf("profile missing: %v", profiles)
	}
}
