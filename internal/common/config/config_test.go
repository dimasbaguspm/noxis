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
	// The listen address and the state file are constants: a deployment that sets them must not
	// change anything, and this test is the place that says so. The console's origin is not the
	// deployment's business under its old name either.
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

// The console may be served from a host the API has never heard of, so the allowed origin is the
// deployment's answer, not a constant. A trailing slash is what a person types and never what a
// browser sends; it is normalised away rather than silently never matching.
func TestConsoleOriginsAreTheDeployments(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"TOKENS":          `{"ops":{"secret":"sk_ops_0123456789abcdef0123456789aa","scopes":["read"]}}`,
		"CONSOLE_ORIGINS": "https://console.example.com, http://localhost:5173/",
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	want := []string{"https://console.example.com", "http://localhost:5173"}
	if len(cfg.ConsoleOrigins) != len(want) {
		t.Fatalf("origins = %v, want %v", cfg.ConsoleOrigins, want)
	}
	for i := range want {
		if cfg.ConsoleOrigins[i] != want[i] {
			t.Fatalf("origins = %v, want %v", cfg.ConsoleOrigins, want)
		}
	}
}

func TestConsoleOriginsDefaultToTheDevConsole(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"TOKENS": `{"ops":{"secret":"sk_ops_0123456789abcdef0123456789aa","scopes":["read"]}}`,
	}))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.ConsoleOrigins) != 2 {
		t.Fatalf("origins = %v, want the two development origins", cfg.ConsoleOrigins)
	}
}

// An origin with a path is a typo: the browser sends scheme://host[:port] and nothing else, so the
// rule would never match and the console would fail with a CORS error instead of a message.
func TestConsoleOriginsRefuseAPath(t *testing.T) {
	for _, raw := range []string{"https://console.example.com/app", "console.example.com", "ftp://x"} {
		if _, err := ParseOrigins(raw); err == nil {
			t.Fatalf("ParseOrigins(%q) accepted a non-origin", raw)
		}
	}
	if _, err := ParseOrigins(" , "); err == nil {
		t.Fatal("ParseOrigins accepted an empty list")
	}
}
