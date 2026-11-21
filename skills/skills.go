// Package skills carries the agent skills this build was tested with.
//
// They ship inside the binary for one reason: a skill that describes a different version than the CLI it
// is used against is worse than no skill, because the agent follows the instructions it was given and the
// binary refuses them. `noxis skills` prints the copy that belongs to the build in hand.
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// The documents. A skill is a directory with a SKILL.md, which is the format pi, Claude Code and Cursor
// all read; README.md explains the set to a person browsing the repository.
//
//go:embed README.md */SKILL.md
var files embed.FS

// Skill is one document: the name it is addressed by, and the frontmatter line that says when to use it.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Lines       int    `json:"lines"`
}

// List reads every skill the binary carries, in name order.
func List() ([]Skill, error) {
	paths, err := fs.Glob(files, "*/SKILL.md")
	if err != nil {
		return nil, err
	}
	all := make([]Skill, 0, len(paths))
	for _, path := range paths {
		body, err := files.ReadFile(path)
		if err != nil {
			return nil, err
		}
		name, description := frontmatter(string(body))
		if name == "" {
			return nil, fmt.Errorf("skills: %s has no name in its frontmatter", path)
		}
		all = append(all, Skill{Name: name, Description: description, Lines: countLines(string(body))})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all, nil
}

// Read answers one document whole, frontmatter included: it is what an agent loads, and rewriting it
// would be inventing a second copy of the skill that nobody tested.
func Read(name string) (string, error) {
	body, err := files.ReadFile(name + "/SKILL.md")
	if err != nil {
		available, listErr := List()
		if listErr != nil {
			return "", err
		}
		names := make([]string, 0, len(available))
		for _, skill := range available {
			names = append(names, skill.Name)
		}
		return "", fmt.Errorf("no skill named %q (have: %s)", name, strings.Join(names, ", "))
	}
	return string(body), nil
}

// frontmatter pulls the two lines a skill is addressed by out of the YAML block at its head. It reads the
// two keys it needs and ignores the rest, because a complete YAML parser is a dependency this does not earn.
func frontmatter(document string) (name, description string) {
	lines := strings.Split(document, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		switch {
		case strings.HasPrefix(line, "name:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		case strings.HasPrefix(line, "description:"):
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return name, description
}

func countLines(document string) int {
	return strings.Count(strings.TrimSuffix(document, "\n"), "\n") + 1
}
