// Package arch_test enforces the dependency direction stated in docs/agent/architecture.md.
// It reads `go list` rather than source, so it fails on the import that actually landed,
// not on the one somebody meant to write.
package arch_test

import (
	"os/exec"
	"strings"
	"testing"
)

// layer is one directory of the tree and the noxis packages it may import. Anything
// outside this module (the standard library, a driver) is always allowed.
type layer struct {
	prefix  string
	allowed []string
}

var layers = []layer{
	// domain is the centre: it names no other package in this module.
	{prefix: "noxis/internal/domain", allowed: []string{"noxis/internal/constants"}},
	{prefix: "noxis/internal/ports", allowed: []string{"noxis/internal/domain"}},
	{prefix: "noxis/internal/app", allowed: []string{
		"noxis/internal/domain", "noxis/internal/ports", "noxis/internal/common", "noxis/internal/constants",
		// A use case may use a shared helper from a sibling package; it may not call another use
		// case's Run. `password` is the only helper today.
		"noxis/internal/app/password",
	}},
	{prefix: "noxis/internal/adapters", allowed: []string{
		"noxis/internal/domain", "noxis/internal/ports", "noxis/internal/common", "noxis/internal/constants",
	}},
	{prefix: "noxis/internal/common", allowed: []string{"noxis/internal/constants", "noxis/internal/common"}},
	{prefix: "noxis/internal/client", allowed: []string{"noxis/internal/constants"}},
	// The transport may call use cases and read the wire vocabulary; it may not name an adapter —
	// that is the composition root's job, and the root lives in cmd/noxis.
	{prefix: "noxis/internal/httpapi", allowed: []string{
		"noxis/internal/domain", "noxis/internal/ports", "noxis/internal/app",
		"noxis/internal/common", "noxis/internal/constants", "noxis/internal/httpapi",
	}},
}

func TestImportDirection(t *testing.T) {
	imports := packageImports(t)
	for _, l := range layers {
		for pkg, deps := range imports {
			if pkg != l.prefix && !strings.HasPrefix(pkg, l.prefix+"/") {
				continue
			}
			for _, dep := range deps {
				if !strings.HasPrefix(dep, "noxis/") || dep == pkg {
					continue
				}
				if !allows(l.allowed, dep) {
					t.Errorf("%s imports %s: outside its layer, which may import %q", pkg, dep, l.allowed)
				}
			}
		}
	}
}

// TestNothingImportsCmd keeps the transports unimportable: the CLI talks HTTP and the
// console will talk HTTP, so no package may reach into a command for a helper.
func TestNothingImportsCmd(t *testing.T) {
	for pkg, deps := range packageImports(t) {
		if strings.HasPrefix(pkg, "noxis/cmd/") {
			continue
		}
		for _, dep := range deps {
			if strings.HasPrefix(dep, "noxis/cmd/") {
				t.Errorf("%s imports %s: a transport is not a library", pkg, dep)
			}
		}
	}
}

func allows(allowed []string, dep string) bool {
	for _, a := range allowed {
		if dep == a || strings.HasPrefix(dep, a+"/") {
			return true
		}
	}
	return false
}

// packageImports returns every package in the module and the noxis packages it imports.
func packageImports(t *testing.T) map[string][]string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", `{{.ImportPath}}|{{join .Imports " "}}`, "./...")
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	imports := make(map[string][]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pkg, deps, _ := strings.Cut(line, "|")
		imports[pkg] = strings.Fields(deps)
	}
	if len(imports) == 0 {
		t.Fatal("go list returned no package")
	}
	return imports
}
