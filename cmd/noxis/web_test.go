package main

import (
	"bytes"
	"noxis/internal/common/dotconfig"
	"strings"
	"testing"
)

// The console may be served from a host the API has never heard of, so the address it calls cannot
// be a default: a page that asks the reader's own machine for the API is a page that fails on every
// deployment but the developer's.
func TestWebServeNeedsTheAPI(t *testing.T) {
	cmd := newWeb()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("web serve started without an API")
	}
	if !strings.Contains(err.Error(), "--api") {
		t.Fatalf("error = %q, want it to name --api", err)
	}
}

// A machine that signed the CLI in is a machine that has an API and a token already: the console
// starts from that session rather than asking for what the terminal knows.
func TestConsoleInheritsTheCLISession(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{APIURL: "https://api.example.com", Token: "tok"})

	session, err := cliSession("", false)
	if err != nil {
		t.Fatalf("cliSession: %v", err)
	}
	if session.APIURL != "https://api.example.com" || session.Token != "tok" {
		t.Fatalf("session = %+v", session)
	}
}

// A machine that never signed in is not an error: the console asks for a token, as it always did.
func TestConsoleWithoutASessionIsNotAnError(t *testing.T) {
	isolated(t)
	session, err := cliSession("", false)
	if err != nil {
		t.Fatalf("cliSession: %v", err)
	}
	if session.APIURL != "" || session.Token != "" {
		t.Fatalf("session = %+v, want the empty one", session)
	}
}

// A profile that was named and is not there is a typo, and a typo has to fail rather than silently
// fall back to the current session.
func TestConsoleRefusesANamedProfileThatIsMissing(t *testing.T) {
	isolated(t)
	if _, err := cliSession("nope", true); err == nil {
		t.Fatal("a named profile that does not exist was accepted")
	}
}
