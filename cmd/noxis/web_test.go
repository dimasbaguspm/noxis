package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/config"
	"noxis/internal/common/dotconfig"
	"strings"
	"testing"
)

// The console may be served from a host the API has never heard of, so the address it calls cannot
// be a default: a page that asks the reader's own machine for the API is a page that fails on every
// deployment but the developer's.
func TestWebServeNeedsTheAPI(t *testing.T) {
	// A machine whose CLI signs in has an API to inherit, so this test has to be the machine that
	// never signed in — otherwise `web serve` finds a session and listens until the test times out.
	isolated(t)
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

// A container has no CLI session, so the environment is what the console signs in with; a machine that
// set neither keeps the profile it already had.
func TestConsoleSessionComesFromTheEnvironment(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{APIURL: "https://stored.example.com", Token: "stored"})

	env := config.ConsoleEnv{Host: "https://api.example.com", Token: "from-env"}
	session, err := consoleSession(env, "", false)
	if err != nil {
		t.Fatalf("consoleSession: %v", err)
	}
	if session.APIURL != env.Host || session.Token != env.Token {
		t.Fatalf("session = %+v, want the environment", session)
	}

	// Nothing in the environment: the stored profile is the session, as it was before.
	session, err = consoleSession(config.ConsoleEnv{}, "", false)
	if err != nil {
		t.Fatalf("consoleSession without an environment: %v", err)
	}
	if session.Token != "stored" || session.APIURL != "https://stored.example.com" {
		t.Fatalf("session = %+v, want the stored profile", session)
	}
}

// A token the API refuses stops the console from listening. A name this process cannot reach does not:
// the browser may reach an address the container cannot, and that is not evidence the token is wrong.
func TestConsoleRefusesATokenTheAPIRejects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"no"}}`))
	}))
	defer server.Close()

	err := prove(context.Background(), server.URL, "super-secret-token")
	if err == nil {
		t.Fatal("a token the API refuses started the console")
	}
	if !strings.Contains(err.Error(), server.URL) || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("error = %q, want it to name the API and the code", err)
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatalf("error = %q, it printed the token", err)
	}

	if err := prove(context.Background(), "http://127.0.0.1:1", "super-secret-token"); err != nil {
		t.Fatalf("an unreachable API stopped the console: %v", err)
	}
}
