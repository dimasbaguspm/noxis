package commands

import (
	"bytes"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
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

// Where a sign-in points, and why. A named profile that already exists wins over the environment, so
// signing in to a second server does not depend on remembering a flag; the flag still wins over all of it.
func TestSigninURLPrecedence(t *testing.T) {
	cfg := dotconfig.Config{
		Current: "staging",
		Profiles: map[string]dotconfig.Profile{
			"staging": {APIURL: "https://staging.example"},
			"prod":    {APIURL: "https://prod.example"},
		},
	}
	for _, tc := range []struct {
		name     string
		flag     string
		env      string
		profile  string
		want     string
		wantFrom string
	}{
		{"the flag wins", "https://flag.example", "https://env.example", "prod", "https://flag.example", ""},
		{"a named profile", "", "https://env.example", "prod", "https://prod.example", `profile "prod"`},
		{"the environment", "", "https://env.example", "unknown", "https://env.example", "$API_URL"},
		{"the current profile", "", "", "unknown", "https://staging.example", `the current profile "staging"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, from := signinURL(tc.flag, cfg, tc.env, tc.profile)
			if got != tc.want || from != tc.wantFrom {
				t.Fatalf("signinURL = (%q, %q), want (%q, %q)", got, from, tc.want, tc.wantFrom)
			}
		})
	}

	empty := dotconfig.Config{Profiles: map[string]dotconfig.Profile{}}
	got, from := signinURL("", empty, "", "prod")
	if got != constants.DefaultAPIURL || from != "the local default" {
		t.Fatalf("signinURL with nothing stored = (%q, %q)", got, from)
	}
}

// The shape a person reaches for first is `signin <token> <host>`. A bare "accepts 1 arg(s)" teaches
// nothing, so the message names the flag that takes a host.
func TestSigninArgsNamesTheFlag(t *testing.T) {
	if err := signinArgs(nil, []string{"tok"}); err != nil {
		t.Fatalf("one token was refused: %v", err)
	}
	for _, args := range [][]string{{}, {""}} {
		if err := signinArgs(nil, args); err == nil || !strings.Contains(err.Error(), "--api-url") {
			t.Fatalf("args %q: err = %v, want it to name --api-url", args, err)
		}
	}
	err := signinArgs(nil, []string{"tok", "https://api.example.com"})
	if err == nil || !strings.Contains(err.Error(), "--api-url") {
		t.Fatalf("two args: err = %v, want it to name --api-url", err)
	}
}
