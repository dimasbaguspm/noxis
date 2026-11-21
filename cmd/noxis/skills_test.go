package main

import (
	"bytes"
	"noxis/cmd/noxis/commands"
	"noxis/internal/constants"
	"strings"
	"testing"
)

// The skills ship inside the binary so that the instructions an agent follows and the CLI that answers it
// are the same build: this is the command that hands them over.
func TestSkillsListsWhatThisBuildCarries(t *testing.T) {
	isolated(t)
	out := &bytes.Buffer{}
	cmd := newSkills(&commands.Session{})
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills: %v", err)
	}
	for _, want := range []string{"noxis-read", "noxis-propose", "noxis-review", "noxis-setup", constants.Build()} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("skills printed %q, want it to mention %q", out.String(), want)
		}
	}
}

// One skill answers the document whole, frontmatter first: it is redirected straight into a skills folder,
// so anything the command adds would corrupt it.
func TestSkillsPrintsOneDocumentVerbatim(t *testing.T) {
	isolated(t)
	out := &bytes.Buffer{}
	cmd := newSkills(&commands.Session{})
	cmd.SetOut(out)
	cmd.SetArgs([]string{"noxis-review"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("skills noxis-review: %v", err)
	}
	if !strings.HasPrefix(out.String(), "---\nname: noxis-review\n") {
		t.Fatalf("document does not start with the frontmatter it was stored with:\n%s", out.String())
	}
	if strings.Contains(out.String(), "this build:") {
		t.Fatal("the single-skill output carries the list's footer, which would land in the file")
	}
}
