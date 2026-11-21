package commands

import (
	"strings"
	"testing"
)

// What a person types at a terminal: a name from the table, or enough of the reference to be unmistakable.
// Everything ambiguous is refused, because the alternative is changing the wrong instance quietly.
func TestATypedValueResolves(t *testing.T) {
	all := []candidate{
		{ref: "01a1fd59-6d1e-7e0d-8784-de451d9156c7", name: "snaptojson-test"},
		{ref: "01a1fd59-8f89-7491-ab10-aabdcfc97f17", name: "snaptojson-prod"},
		{ref: "01b2fd59-6d1e-7e0d-8784-de451d9156c7", name: "lololo"},
	}

	for _, tc := range []struct {
		name  string
		typed string
		want  string
		fails bool
		says  string
	}{
		{"the name as printed", "snaptojson-test", "01a1fd59-6d1e-7e0d-8784-de451d9156c7", false, ""},
		{"a name typed in caps", "LOLOLO", "01b2fd59-6d1e-7e0d-8784-de451d9156c7", false, ""},
		{"a unique prefix", "01b2", "01b2fd59-6d1e-7e0d-8784-de451d9156c7", false, ""},
		{"a longer unique prefix", "01b2fd59-6d1e", "01b2fd59-6d1e-7e0d-8784-de451d9156c7", false, ""},
		// Two targets share it: the error has to say which, not pick one.
		{"an ambiguous prefix", "01a1", "", true, "matches 2 targets"},
		{"nothing at all", "nope", "", true, "no target answers to"},
		{"empty", "", "", true, "name a target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := match("target", tc.typed, all)
			if !tc.fails {
				if err != nil {
					t.Fatalf("match(%q): %v", tc.typed, err)
				}
				if got != tc.want {
					t.Fatalf("match(%q) = %q, want %q", tc.typed, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("match(%q) resolved to %q, want a refusal", tc.typed, got)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("error = %q, want it to say %q", err, tc.says)
			}
		})
	}
}

// An ambiguous prefix lists what it matched, with enough of each reference to type the rest of it.
func TestAnAmbiguousPrefixNamesItsCandidates(t *testing.T) {
	_, err := match("target", "01a1", []candidate{
		{ref: "01a1fd59-6d1e-7e0d-8784-de451d9156c7", name: "snaptojson-test"},
		{ref: "01a1fd59-8f89-7491-ab10-aabdcfc97f17", name: "snaptojson-prod"},
	})
	if err == nil {
		t.Fatal("an ambiguous prefix resolved to one target")
	}
	for _, want := range []string{"snaptojson-test (01a1fd59)", "snaptojson-prod (01a1fd59)"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to name %q", err, want)
		}
	}
}

// A proposal has no name of its own, so its title is what a person matches on, and a prefix of the
// reference otherwise.
func TestAProposalPrefixResolves(t *testing.T) {
	all := []candidate{
		{ref: "01a2aa59-6d1e-7e0d-8784-de451d9156c7", name: "drop the legacy index"},
		{ref: "01a2bb59-6d1e-7e0d-8784-de451d9156c7", name: "add email to users"},
	}
	if got, err := match("proposal", "01a2aa", all); err != nil || got != all[0].ref {
		t.Fatalf("prefix: %q, %v", got, err)
	}
	if got, err := match("proposal", "add email to users", all); err != nil || got != all[1].ref {
		t.Fatalf("title: %q, %v", got, err)
	}
}

// A name wins over a prefix: a target called "prod" is not ambiguous because "prod-old" also exists.
func TestANameBeatsAPrefix(t *testing.T) {
	all := []candidate{
		{ref: "01b2aa59-6d1e-7e0d-8784-de451d9156c7", name: "prod"},
		{ref: "01b2bb59-6d1e-7e0d-8784-de451d9156c7", name: "prod-old"},
	}
	if got, err := match("target", "prod", all); err != nil || got != all[0].ref {
		t.Fatalf("match = %q, %v; want the target actually called prod", got, err)
	}
}
