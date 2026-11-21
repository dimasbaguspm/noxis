package commands

import (
	"sort"
	"strings"
	"testing"
)

// Eight acts on one change, under one noun: the names a reviewer needs before the first proposal is read.
func TestTheProposalGroupHoldsEveryVerb(t *testing.T) {
	group := NewProposal(&Session{})

	got := make([]string, 0, len(group.Commands()))
	for _, child := range group.Commands() {
		got = append(got, child.Name())
	}
	sort.Strings(got)
	want := []string{"apply", "approve", "events", "ls", "rehearse", "reject", "show", "watch"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("proposal holds %v, want %v", got, want)
	}

	// The verbs that take a reference say so, and say that a name works: the whole point of accepting one.
	for _, child := range group.Commands() {
		if child.Name() == "ls" {
			continue
		}
		if !strings.Contains(child.Use, "<ref|name>") {
			t.Fatalf("%s: Use = %q, want it to name the argument", child.Name(), child.Use)
		}
	}
}

// The old top-level spelling still runs and says where it went: a script written last month is not a
// reason to break a deployment, but it should be told.
func TestADeprecatedSpellingSaysWhereItWent(t *testing.T) {
	cmd := Deprecated(NewApprove(&Session{}), "approve <ref>", "proposal approve")

	if cmd.Name() != "approve" {
		t.Fatalf("name = %q, want approve", cmd.Name())
	}
	if cmd.Deprecated == "" || !strings.Contains(cmd.Deprecated, "noxis proposal approve") {
		t.Fatalf("Deprecated = %q, want the replacement named", cmd.Deprecated)
	}
	// The group owns the aliases; a leftover one here would shadow a sibling at the root.
	if len(cmd.Aliases) != 0 {
		t.Fatalf("aliases = %v, want none", cmd.Aliases)
	}
}
