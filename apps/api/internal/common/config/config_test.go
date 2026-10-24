package config

import (
	"testing"
	"time"

	"noxis/internal/constants"
)

// Guards the env contract: defaults make an empty environment runnable, and the secret key is always required.
func TestLoadConfigDefaultsAndValidate(t *testing.T) {
	for _, e := range []string{EnvListen, EnvDB, EnvSecretKey} {
		t.Setenv(e, "")
	}

	cfg := LoadConfig()
	if cfg.Listen != defaultListen || cfg.DB != defaultDB {
		t.Fatalf("defaults not applied: listen=%q db=%q", cfg.Listen, cfg.DB)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a missing secret key must fail Validate")
	}

	t.Setenv(EnvListen, ":1234")
	t.Setenv(EnvDB, "/tmp/noxis.db")
	cfg = LoadConfig()
	if cfg.Listen != ":1234" || cfg.DB != "/tmp/noxis.db" {
		t.Fatalf("env override not applied: listen=%q db=%q", cfg.Listen, cfg.DB)
	}

	t.Setenv(EnvSecretKey, "0000")
	if err := (LoadConfig()).Validate(); err != nil {
		t.Fatalf("a set secret key must pass Validate: %v", err)
	}
}

func TestAccessGrantTTL(t *testing.T) {
	t.Setenv(EnvAccessGrantDays, "")
	if got := LoadConfig().GrantTTL(); got != constants.AccessGrantTTL {
		t.Fatalf("default ttl = %v, want %v", got, constants.AccessGrantTTL)
	}
	t.Setenv(EnvAccessGrantDays, "30")
	if got := LoadConfig().GrantTTL(); got != 30*24*time.Hour {
		t.Fatalf("configured ttl = %v, want 30 days", got)
	}
	t.Setenv(EnvAccessGrantDays, "not-a-number")
	if got := LoadConfig().GrantTTL(); got != constants.AccessGrantTTL {
		t.Fatalf("invalid ttl must fall back, got %v", got)
	}
}
