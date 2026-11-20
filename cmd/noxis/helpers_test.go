package main

import (
	"noxis/internal/common/dotconfig"
	"testing"
)

// isolated gives a test its own config directory: the CLI's file is the subject here, not the
// developer's own session.
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
