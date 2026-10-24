// Package arch enforces inward-only dependencies and forbids same-layer imports.
package arch

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// layerOf classifies packages by layer and excludes test-only packages.
func layerOf(pkg string) (int, bool) {
	switch {
	case pkg == "noxis/internal/constants":
		return 0, true
	case pkg == "noxis/internal/entity":
		return 1, true
	case pkg == "noxis/internal/interfaces":
		return 2, true
	case pkg == "noxis/internal/common" || strings.HasPrefix(pkg, "noxis/internal/common/"):
		return 3, true
	case pkg == "noxis/internal/repositories" || strings.HasPrefix(pkg, "noxis/internal/repositories/"):
		return 4, true
	case strings.HasPrefix(pkg, "noxis/internal/modules/"):
		return 5, true
	case pkg == "noxis/internal/modules":
		return 6, true
	case pkg == "noxis/internal/usecases":
		return 7, true
	case strings.HasPrefix(pkg, "noxis/internal/usecases/"):
		return 8, true
	case pkg == "noxis/cmd/noxis/server/middlewares" || strings.HasPrefix(pkg, "noxis/cmd/noxis/server/middlewares/"):
		return 9, true
	case pkg == "noxis/cmd/noxis/server" || strings.HasPrefix(pkg, "noxis/cmd/noxis/server/"):
		return 10, true
	}
	return 0, false
}

const compositionRoot = "noxis/cmd/noxis"

type pkg struct {
	ImportPath string
	Imports    []string
}

func listPackages(t *testing.T) []pkg {
	t.Helper()
	out, err := exec.Command("go", "list", "-json", "noxis/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var pkgs []pkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func TestLayering(t *testing.T) {
	for _, p := range listPackages(t) {
		if p.ImportPath == compositionRoot {
			continue // process entry: it wires everything, exempt by design
		}
		from, ok := layerOf(p.ImportPath)
		if !ok {
			continue // test-only packages are outside the model
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, "noxis/") || imp == compositionRoot {
				continue
			}
			to, ok := layerOf(imp)
			if !ok {
				continue
			}
			switch {
			case to == from:
				t.Errorf("%s imports its sibling %s (both layer %d): same-layer packages must not import each other",
					p.ImportPath, imp, from)
			case to > from:
				t.Errorf("%s (layer %d) imports %s (layer %d): imports must point down the layers",
					p.ImportPath, from, imp, to)
			}
		}
	}
}
