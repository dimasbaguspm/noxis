package main

import (
	"bytes"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
	"strings"
	"testing"
	"time"
)

// A bare `noxis` is the first thing a person types, so it has to say where it points and what to do
// next — without a token, without a network call, and without failing because nobody signed in.
func TestBareNoxisGreets(t *testing.T) {
	isolated(t)

	root := newRoot()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(nil)
	if err := root.Execute(); err != nil {
		t.Fatalf("bare noxis: %v (stderr %s)", err, errOut.String())
	}

	greeting := out.String()
	for _, want := range []string{"http serve", "web serve", "auth signin", "noxis help", "--json", "Not signed in"} {
		if !strings.Contains(greeting, want) {
			t.Fatalf("welcome does not mention %q:\n%s", want, greeting)
		}
	}
}

// A signed-in profile is named in the greeting, and the token is not.
func TestWelcomeNamesTheSession(t *testing.T) {
	isolated(t)
	store(t, dotconfig.Profile{
		APIURL: "http://staging", Token: "hunter2", Caller: "reviewer",
		Scopes: []string{"read", "approve"}, CheckedAt: time.Now().UTC(),
	})

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(nil)
	if err := root.Execute(); err != nil {
		t.Fatalf("bare noxis: %v", err)
	}
	if !strings.Contains(out.String(), "Signed in to http://staging as reviewer") {
		t.Fatalf("welcome = %s", out.String())
	}
	if strings.Contains(out.String(), "hunter2") {
		t.Fatal("the welcome printed the token")
	}
}

// `?` and `help` are the same map, and it is grouped by what a person is trying to do.
func TestHelpIsGrouped(t *testing.T) {
	isolated(t)

	for _, invocation := range [][]string{{"help"}, {"?"}} {
		root := newRoot()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(invocation)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", invocation, err)
		}
		text := out.String()
		for _, want := range []string{"SESSION", "THIS BINARY", "TARGETS", "READING", "PROPOSALS", "apply"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%v does not print %q:\n%s", invocation, want, text)
			}
		}
	}
}

// `noxis help <command>` is the command's own usage, which is what a person typing it expects.
func TestHelpForOneCommand(t *testing.T) {
	isolated(t)

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"help", "target"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help target: %v", err)
	}
	if !strings.Contains(out.String(), "noxis target") {
		t.Fatalf("help target = %s", out.String())
	}
}

// The console refuses a port it cannot bind before it starts listening, and says which number was
// wrong rather than failing inside a goroutine.
func TestConsoleRefusesAnImpossiblePort(t *testing.T) {
	isolated(t)

	root := newRoot()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"web", "serve", "--port", "70000"})
	if err := root.Execute(); err == nil {
		t.Fatal("a port outside 1-65535 was accepted")
	}
}

// The serve command is honest about what it is serving.
func TestBuiltSaysWhichConsole(t *testing.T) {
	if !strings.Contains(built(true), "built into this binary") {
		t.Fatalf("built(true) = %q", built(true))
	}
	if !strings.Contains(built(false), "placeholder") {
		t.Fatalf("built(false) = %q", built(false))
	}
}

// A version is a fact about the binary, not about a session: a container that has not been signed
// in still has to be able to ask what it is running.
func TestVersionNeedsNoSession(t *testing.T) {
	isolated(t)

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version without a session: %v", err)
	}
	if !strings.Contains(out.String(), constants.Version) {
		t.Fatalf("version = %q", out.String())
	}
}

// A command that is built but never registered is a command nobody can run: `watch` shipped that way
// once, compiled and tested and unreachable. The grouping is the list of commands a person is meant to
// reach, so it is also the list every registration is checked against.
func TestEveryGroupedCommandIsRegistered(t *testing.T) {
	known := map[string]bool{}
	for _, command := range newRoot().Commands() {
		known[command.Name()] = true
		for _, alias := range command.Aliases {
			known[alias] = true
		}
	}
	for _, group := range groups {
		for _, name := range group.commands {
			if !known[name] {
				t.Errorf("%q is grouped under %q but nothing registers it", name, group.title)
			}
		}
	}
}

// Every other Go CLI answers `--version`, and a version probe reaches for it before it reads a help page.
// It prints the same two lines `noxis version` does, and needs no session to do it.
func TestVersionFlagAnswers(t *testing.T) {
	isolated(t)

	root := newRoot()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--version: %v (stderr %s)", err, errOut.String())
	}

	got := out.String()
	for _, want := range []string{constants.Build(), constants.ReleaseURL} {
		if !strings.Contains(got, want) {
			t.Fatalf("--version printed %q, want it to mention %q", got, want)
		}
	}
}
