package commands

import (
	"bytes"
	"noxis/internal/common/dotconfig"
	"strings"
	"testing"
)

// Two servers, two tokens already on disk: switching between them is a command, not a re-signin.
func TestAuthUseSwitchesTheProfile(t *testing.T) {
	isolated(t)
	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("default", dotconfig.Profile{APIURL: "https://one.example", Token: "t1", Caller: "ops"})
	cfg.Put("staging", dotconfig.Profile{APIURL: "https://two.example", Token: "t2", Caller: "agent"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	out := &bytes.Buffer{}
	use := newUse()
	use.SetOut(out)
	use.SetArgs([]string{"staging"})
	if err := use.Execute(); err != nil {
		t.Fatalf("use staging: %v", err)
	}

	reloaded, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if reloaded.Current != "staging" {
		t.Fatalf("current = %q, want staging", reloaded.Current)
	}
	if !strings.Contains(out.String(), "https://two.example") {
		t.Fatalf("use printed %q, want it to name the server it switched to", out.String())
	}
}

// A typo names the profiles that exist: the answer is on this machine, so there is no reason to make
// someone run a second command to find it.
func TestAuthUseRefusesAnUnknownProfile(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{APIURL: "https://one.example", Token: "t1"})

	use := newUse()
	use.SetOut(&bytes.Buffer{})
	use.SetArgs([]string{"nope"})
	err := use.Execute()
	if err == nil {
		t.Fatal("use accepted a profile that does not exist")
	}
	if !strings.Contains(err.Error(), "default") {
		t.Fatalf("error = %q, want it to list the stored profiles", err)
	}
}

// Which sessions this machine holds, and which one a bare command would use: the star is the answer.
func TestAuthLsMarksTheCurrentProfile(t *testing.T) {
	isolated(t)
	cfg, err := dotconfig.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Put("staging", dotconfig.Profile{APIURL: "https://two.example", Token: "t2", Caller: "agent"})
	cfg.Put("default", dotconfig.Profile{APIURL: "https://one.example", Token: "t1", Caller: "ops"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := cfg.Use("default"); err != nil {
		t.Fatalf("use: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	out := &bytes.Buffer{}
	ls := newLs()
	ls.SetOut(out)
	if err := ls.Execute(); err != nil {
		t.Fatalf("ls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("ls printed %d lines, want one per profile:\n%s", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "* default") {
		t.Fatalf("first line = %q, want the current profile marked", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  staging") {
		t.Fatalf("second line = %q, want staging unmarked and in name order", lines[1])
	}
}
