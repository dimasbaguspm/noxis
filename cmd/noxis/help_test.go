package main

import (
	"bytes"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
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
	for _, want := range []string{"http serve", "auth signin", "noxis help", "--json", "profiles: none", constants.Build()} {
		if !strings.Contains(greeting, want) {
			t.Fatalf("welcome does not mention %q:\n%s", want, greeting)
		}
	}
}

// A signed-in profile is named in the greeting, and the token is not.
func TestWelcomeNamesTheSession(t *testing.T) {
	isolated(t)
	// A closed port: the server line has to give up in milliseconds, not wait out its timeout.
	store(t, dotconfig.Profile{
		APIURL: "http://127.0.0.1:1", Token: "hunter2", Caller: "reviewer",
		Scopes: []string{"read", "approve"}, CheckedAt: time.Now().UTC(),
	})

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(nil)
	if err := root.Execute(); err != nil {
		t.Fatalf("bare noxis: %v", err)
	}
	greeting := out.String()
	for _, want := range []string{"profiles:", "* default", "http://127.0.0.1:1", "reviewer", "did not answer"} {
		if !strings.Contains(greeting, want) {
			t.Fatalf("welcome does not mention %q:\n%s", want, greeting)
		}
	}
	if strings.Contains(greeting, "hunter2") {
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

// Two commands answering to the same name — or one command whose alias steals another's name — is a list
// nobody can read: the same word appears twice and one of the two is unreachable. Cobra quietly adds its
// own `help` unless it is told which one to use, which is exactly how this happened once.
func TestNoTwoCommandsAnswerToTheSameName(t *testing.T) {
	isolated(t)
	root := newRoot()
	// The defaults are what a real run adds, so the test asks for them too: a rule checked before cobra
	// has added its own commands is a rule that misses the case it was written for.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	var walk func(parent *cobra.Command)
	walk = func(parent *cobra.Command) {
		seen := map[string]string{}
		for _, child := range parent.Commands() {
			for _, name := range append([]string{child.Name()}, child.Aliases...) {
				if other, taken := seen[name]; taken && other != child.Name() {
					t.Errorf("`%s`: %q is answered by both %q and %q",
						parent.CommandPath(), name, other, child.Name())
				}
				seen[name] = child.Name()
			}
			walk(child)
		}
	}
	walk(root)

	help := 0
	for _, child := range root.Commands() {
		if child.Name() == "help" {
			help++
		}
	}
	if help != 1 {
		t.Fatalf("%d commands are called `help`; the tree must have exactly one", help)
	}
}
