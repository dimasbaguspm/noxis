package dotconfig_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"noxis/internal/common/dotconfig"
)

// A file that holds a bearer token is only as private as its mode, and a directory that lists it is
// just as loud. Both are checked through the real file, not through the struct.
func TestSaveKeepsTheFilePrivate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("default", dotconfig.Profile{APIURL: "http://localhost:9000", Token: "s3cret"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	path, err := dotconfig.Path()
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("config file mode = %o, want 600", mode)
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if mode := dir.Mode().Perm(); mode != 0o700 {
		t.Fatalf("config dir mode = %o, want 700", mode)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	checked := time.Now().UTC().Truncate(time.Second)
	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("staging", dotconfig.Profile{
		APIURL: "https://staging:9000", Token: "tok",
		Caller: "agent", Scopes: []string{"read", "propose"}, CheckedAt: checked,
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	again, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again.Current != "staging" {
		t.Fatalf("current = %q, want staging", again.Current)
	}
	stored, err := again.Profile("staging")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if stored.Token != "tok" || stored.Caller != "agent" || stored.APIURL != "https://staging:9000" {
		t.Fatalf("round trip lost a field: %+v", stored)
	}
	if !stored.CheckedAt.Equal(checked) {
		t.Fatalf("checked_at = %s, want %s", stored.CheckedAt, checked)
	}
}

// Forgetting the current profile has to leave nothing behind: a "signed out" file that still names a
// token is the failure mode this command exists to prevent.
func TestForgetRemovesTheToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("default", dotconfig.Profile{APIURL: "http://localhost:9000", Token: "s3cret"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := cfg.Forget("default"); err != nil {
		t.Fatalf("forget: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	again, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again.Current != "" {
		t.Fatalf("current = %q after signout, want empty", again.Current)
	}
	if _, err := again.Profile("default"); !errors.Is(err, dotconfig.ErrNoProfile) {
		t.Fatalf("profile after forget: %v, want ErrNoProfile", err)
	}

	raw, err := os.ReadFile(again.FilePath())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if containsSecret(string(raw)) {
		t.Fatal("the token is still in the file after signout")
	}
}

// Reading a file that was never written is how every session starts: it must not be an error.
func TestLoadWithoutAFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Current != "" || len(cfg.Profiles) != 0 {
		t.Fatalf("empty file read as %+v", cfg)
	}
	if _, err := cfg.Profile("default"); !errors.Is(err, dotconfig.ErrNoProfile) {
		t.Fatalf("profile in an empty config: %v, want ErrNoProfile", err)
	}
}

func containsSecret(raw string) bool {
	return len(raw) > 0 && (contains(raw, "s3cret") || contains(raw, `"token": "`))
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
