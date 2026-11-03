package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// moduleVerbs is the verb set each family declares in its own files. Every module inherits the rest from
// interfaces.Unsupported, which answers with ErrUnsupported, so a verb missing here still compiles — it just refuses a
// call the driver meant to serve, which is exactly what this test catches.
var moduleVerbs = map[string][]string{
	"postgres":  {"Query", "DryRun", "Analyze", "Tables", "Rows", "Referenced", "ServerInfo", "Kind", "VerifyWrite", "ValidateSQLChange", "ApplySQL"},
	"mysql":     {"Query", "DryRun", "Analyze", "Tables", "Rows", "Referenced", "ServerInfo", "Kind", "VerifyWrite", "ValidateSQLChange", "ApplySQL"},
	"mariadb":   {"Query", "DryRun", "Analyze", "Tables", "Rows", "Referenced", "ServerInfo", "Kind", "VerifyWrite", "ValidateSQLChange", "ApplySQL"},
	"sqlite":    {"Query", "DryRun", "Analyze", "Tables", "Rows", "Referenced", "ServerInfo", "Kind", "VerifyWrite", "ValidateSQLChange", "ApplySQL"},
	"cockroach": {"Query", "DryRun", "Analyze", "Tables", "Rows", "Referenced", "ServerInfo", "Kind", "VerifyWrite", "ValidateSQLChange", "ApplySQL"},

	"redis":     {"Keys", "Value", "Delete", "Flush", "Expire", "Info"},
	"valkey":    {"Keys", "Value", "Delete", "Flush", "Expire", "Info"},
	"dragonfly": {"Keys", "Value", "Delete", "Flush", "Expire", "Info"},

	"mongodb": {"Databases", "Collections", "Run", "ValidateRequest", "ApplyDocument", "IsWrite"},
	"couchdb": {"Databases", "Collections", "Run", "ValidateRequest", "ApplyDocument", "IsWrite"},

	"elasticsearch": {"Targets", "Search", "ValidateSearchChange", "ApplySearch", "IsWrite"},
	"meilisearch":   {"Targets", "Search", "ValidateSearchChange", "ApplySearch", "IsWrite"},
	"splunk":        {"Targets", "Search", "ValidateSearchChange", "ApplySearch", "IsWrite"},
}

// engineVerbs are the identity methods every module declares for itself; Unsupported deliberately does not answer them.
var engineVerbs = []string{"Driver", "Supports", "Capabilities", "Ping"}

// TestEachModuleDeclaresItsVerbs proves a module implements the verbs its family serves instead of inheriting a refusal:
// the declaration is read from the package's own files, so a promoted Unsupported method does not count.
func TestEachModuleDeclaresItsVerbs(t *testing.T) {
	for name, familyVerbs := range moduleVerbs {
		declared := declaredMethods(t, filepath.Join(moduleRoot, name))
		for _, verb := range append(slices.Clone(familyVerbs), engineVerbs...) {
			if !declared[verb] {
				t.Errorf("module %s does not declare %s: it would answer Unsupported", name, verb)
			}
		}
		for verb := range declared {
			if slices.Contains(familyVerbs, verb) || slices.Contains(engineVerbs, verb) {
				continue
			}
			t.Errorf("module %s declares %s, which belongs to another family", name, verb)
		}
	}
}

// declaredMethods reads the methods a package declares on the module type (*DB) in its own non-test files.
func declaredMethods(t *testing.T, dir string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	methods := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			// Only the exported surface is the contract; the helpers a module keeps for itself answer nobody.
			if fn.Name.IsExported() && isModuleReceiver(fn.Recv.List[0].Type) {
				methods[fn.Name.Name] = true
			}
		}
	}
	return methods
}

// isModuleReceiver reports whether a receiver type is the module's own *DB.
func isModuleReceiver(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "DB"
}
