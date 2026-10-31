package arch

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// moduleRoot is where the engine adapters live, relative to this test package.
const moduleRoot = "../../internal/modules"

// TestNoModuleCopiesAnother fails when two driver packages ship a byte-identical body. That is how the cockroach module
// ended up naming postgres in its queries: a fix landed in one copy and not the other, and the compiler was happy.
//
// A near-identical pair is not flagged — two engines may legitimately read alike. A byte-identical one is a fix waiting
// to be forgotten, so it fails here instead.
func TestNoModuleCopiesAnother(t *testing.T) {
	bodies := map[string][]string{}
	err := filepath.WalkDir(moduleRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		key := hex.EncodeToString(sum[:])
		bodies[key] = append(bodies[key], strings.TrimPrefix(path, moduleRoot+"/"))
		return nil
	})
	if err != nil {
		t.Fatalf("walk modules: %v", err)
	}
	for _, files := range bodies {
		if len(files) < 2 {
			continue
		}
		t.Errorf("two driver packages ship the same body: %s", strings.Join(files, " and "))
	}
}

// TestModulesImportNoSibling restates the layering rule for the engine adapters alone: a module is an engine, so it
// depends on the seams and never on another engine.
func TestModulesImportNoSibling(t *testing.T) {
	for _, p := range listPackages(t) {
		if !strings.HasPrefix(p.ImportPath, "noxis/internal/modules/") {
			continue
		}
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, "noxis/internal/modules/") {
				t.Errorf("%s imports its sibling %s: a module may not depend on another module", p.ImportPath, imp)
			}
		}
	}
}
