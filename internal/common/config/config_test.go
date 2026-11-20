package config

import (
	"os"
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadReadsOnlyWhatItOwns(t *testing.T) {
	// The listen address, the state file and the console origins are constants now: a deployment
	// that sets them must not change anything, and this test is the place that says so.
	cfg, err := LoadFrom(env(map[string]string{
		"TOKENS":       `{"ops":{"secret":"sk_ops_0123456789abcdef0123456789aa","scopes":["read"]}}`,
		"HTTP_ADDR":    ":1234",
		"STATE":        "/tmp/elsewhere.db",
		"CORS_ORIGINS": "https://example.com",
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.Tokens) != 1 || cfg.Tokens[0].Name != "ops" {
		t.Fatalf("tokens = %+v", cfg.Tokens)
	}
	if cfg.QueryTimeout != 30*time.Second || cfg.ProbeTimeout != 5*time.Second {
		t.Fatalf("timeouts = %s / %s", cfg.QueryTimeout, cfg.ProbeTimeout)
	}
	if cfg.LogLevel != "info" || cfg.LogFormat != "text" {
		t.Fatalf("logging = %s / %s", cfg.LogLevel, cfg.LogFormat)
	}
}

func TestLoadRefusesWithoutTokens(t *testing.T) {
	if _, err := LoadFrom(env(map[string]string{})); err == nil {
		t.Fatal("LoadFrom accepted an environment with no token")
	}
}

func TestLoadRefusesABadDuration(t *testing.T) {
	if _, err := LoadFrom(env(map[string]string{
		"TOKENS":        `{"ops":{"secret":"sk_ops_0123456789abcdef0123456789aa","scopes":["read"]}}`,
		"QUERY_TIMEOUT": "soon",
	})); err == nil {
		t.Fatal("LoadFrom accepted a duration it cannot parse")
	}
}

func TestLoadReadsTokensFromAFile(t *testing.T) {
	path := t.TempDir() + "/tokens.json"
	if err := os.WriteFile(path, []byte(`{"ci":{"secret":"sk_ci_0123456789abcdef0123456789aaa","scopes":["read","propose"]}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadFrom(env(map[string]string{"TOKENS_FILE": path}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.Tokens) != 1 || cfg.Tokens[0].Name != "ci" {
		t.Fatalf("tokens = %+v", cfg.Tokens)
	}
}
