package skills

import (
	"strings"
	"testing"
)

// The four skills are the product's operating manual: if one stops being embedded, an agent on a machine
// without the repository has no instructions at all.
func TestTheFourSkillsAreEmbedded(t *testing.T) {
	all, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"noxis-propose", "noxis-read", "noxis-review", "noxis-setup"}
	if len(all) != len(want) {
		t.Fatalf("skills = %d, want %d: %+v", len(all), len(want), all)
	}
	for i, name := range want {
		if all[i].Name != name {
			t.Fatalf("skills[%d] = %q, want %q (the list is in name order)", i, all[i].Name, name)
		}
		if all[i].Description == "" {
			t.Fatalf("%s has no description: an agent picks a skill by that line", name)
		}
		if all[i].Lines < 20 {
			t.Fatalf("%s is %d lines, which is not a document", name, all[i].Lines)
		}
	}
}

// Read answers the document whole: the frontmatter is what an agent loads first, so a body without it is
// not the skill.
func TestReadReturnsTheDocumentWhole(t *testing.T) {
	body, err := Read("noxis-read")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.HasPrefix(body, "---\nname: noxis-read\n") {
		t.Fatalf("document does not start with its frontmatter:\n%s", body[:120])
	}
	if !strings.Contains(body, "noxis target ls") {
		t.Fatal("the document has no commands in it")
	}
}

// A typo answers with the names that exist: the reader is one command away from the right one.
func TestAnUnknownSkillNamesTheOthers(t *testing.T) {
	_, err := Read("noxis-nope")
	if err == nil {
		t.Fatal("Read accepted a skill that is not embedded")
	}
	for _, want := range []string{"noxis-nope", "noxis-read", "noxis-review"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err, want)
		}
	}
}
