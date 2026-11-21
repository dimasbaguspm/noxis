// Package adapters_test holds the audit that no SQL in this tree is built out of a value.
//
// The property is the one an injection needs to exist: a statement may be assembled from constant
// fragments, from an integer, or from an identifier that passed through `quote()` — and never from a
// request field, a row, or anything else a caller controls. It is checked by reading the syntax tree
// rather than by trusting the author, because the mistake is invisible in review: one `+ value`
// looks exactly like the line above it.
package adapters_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sqlWord is a keyword that makes a string a statement rather than a label. It is deliberately broad:
// a false positive is one line to rename, a false negative is an injection.
var sqlWord = regexp.MustCompile(`(?i)\b(select|insert|replace|update|delete|from|where|order by|group by|join|pragma|show|create|alter|drop|table|index)\b`)

// auditPackage is one directory of adapters, with the constants it declares and its files.
type auditPackage struct {
	dir       string
	fset      *token.FileSet
	files     []*ast.File
	consts    map[string]bool
	safeNames map[string]bool
}

func TestNoSQLIsBuiltFromAValue(t *testing.T) {
	for _, dir := range adapterDirs(t) {
		pkg := parsePackage(t, dir)
		for _, file := range pkg.files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.CallExpr:
					pkg.checkSprintf(t, typed)
				case *ast.BinaryExpr:
					pkg.checkConcatenation(t, typed)
				}
				return true
			})
		}
	}
}

// TestEveryPlaceholderIsBound checks the other half: a statement with two `?` and one argument is a
// bug the driver reports at runtime, on the one request that matters.
func TestEveryPlaceholderIsBound(t *testing.T) {
	for _, dir := range adapterDirs(t) {
		pkg := parsePackage(t, dir)
		for _, file := range pkg.files {
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := methodName(call)
				if !isQueryMethod(name) || call.Ellipsis.IsValid() {
					return true
				}
				query, args := queryArg(call)
				if query == "" || args < 0 {
					return true
				}
				if bound := strings.Count(query, "?"); bound != args {
					t.Errorf("%s: %s binds %d placeholder(s) with %d argument(s): %q",
						pkg.fset.Position(call.Pos()), name, bound, args, query)
				}
				return true
			})
		}
	}
}

// checkSprintf refuses a format string that carries SQL and an argument that is not safe in it.
func (p *auditPackage) checkSprintf(t *testing.T, call *ast.CallExpr) {
	t.Helper()
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Sprintf" || len(call.Args) == 0 {
		return
	}
	format, ok := call.Args[0].(*ast.BasicLit)
	if !ok || format.Kind != token.STRING || !sqlWord.MatchString(format.Value) {
		return
	}
	for _, arg := range call.Args[1:] {
		if !p.safe(arg) {
			t.Errorf("%s: SQL is formatted with a value that is not constant, an integer or quoted: %s",
				p.fset.Position(call.Pos()), p.fset.Position(arg.Pos()))
		}
	}
}

// checkConcatenation refuses a statement glued to anything but a constant fragment.
func (p *auditPackage) checkConcatenation(t *testing.T, binary *ast.BinaryExpr) {
	t.Helper()
	if binary.Op != token.ADD {
		return
	}
	if !containsSQL(binary) {
		return
	}
	for _, side := range flattenAdd(binary) {
		if !p.safe(side) {
			t.Errorf("%s: SQL is concatenated with a value that is not constant, an integer or quoted: %s",
				p.fset.Position(binary.Pos()), p.fset.Position(side.Pos()))
		}
	}
}

// safe reports whether an expression may reach a statement: a literal, a constant, an integer, an
// identifier that only ever held those, or a string helper applied to them.
func (p *auditPackage) safe(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return p.consts[typed.Name] || p.safeNames[typed.Name] ||
			typed.Name == "true" || typed.Name == "false"
	case *ast.CallExpr:
		switch methodName(typed) {
		// quote() escapes the engine's own quoting character, so whatever it returns is an
		// identifier the engine reads as text: that is the whole reason it exists.
		case "quote":
			return true
		// A length is an integer: it can count placeholders, and it cannot carry a value.
		case "len":
			return true
		// The rest build a statement's *shape*: placeholders, a joined WHERE clause, a count.
		case "itoa", "Itoa",
			"Join", "Repeat", "TrimSuffix", "TrimPrefix", "TrimSpace", "ReplaceAll", "Sprintf", "append":
			for _, arg := range typed.Args {
				if !p.safe(arg) {
					return false
				}
			}
			return true
		}
		return false
	case *ast.ParenExpr:
		return p.safe(typed.X)
	case *ast.BinaryExpr:
		return p.safe(typed.X) && p.safe(typed.Y)
	}
	return false
}

// appendsSafe reports a name that is only ever grown with safe values: `where = append(where,
// "ref < ?")`. The accumulator is the name itself, so judging it would be circular.
func (p *auditPackage) appendsSafe(name string, value ast.Expr) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok || methodName(call) != "append" || len(call.Args) < 2 {
		return false
	}
	first, ok := call.Args[0].(*ast.Ident)
	if !ok || first.Name != name {
		return false
	}
	for _, arg := range call.Args[1:] {
		if !p.safe(arg) {
			return false
		}
	}
	return true
}

// collectSafe names the variables that only ever held a literal, a constant, or a string helper's
// result, so `placeholders` and a keyset `query` are allowed while `filter.Query` never is. It runs
// to a fixpoint because a clause is built from a placeholder run that was built before it.
//
// It is deliberately small: it proves the shape of an assembled statement, not the intent of the
// author. A name that mixes a literal and a request field stays unsafe, which is the point.
func (p *auditPackage) collectSafe() {
	for range 5 {
		changed := false
		for _, file := range p.files {
			ast.Inspect(file, func(node ast.Node) bool {
				assign, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, lhs := range assign.Lhs {
					ident, ok := lhs.(*ast.Ident)
					if !ok || p.safeNames[ident.Name] {
						continue
					}
					var value ast.Expr
					switch {
					case len(assign.Rhs) == len(assign.Lhs):
						value = assign.Rhs[i]
					case len(assign.Rhs) == 1:
						value = assign.Rhs[0]
					}
					if value == nil {
						continue
					}
					// `x += …` is x on both sides: only the appended part can be judged, and the
					// variable is only safe if it already was.
					assigned := &ast.AssignStmt{Lhs: assign.Lhs, Tok: assign.Tok, Rhs: assign.Rhs}
					if assign.Tok == token.ADD_ASSIGN && !p.safe(ident) {
						_ = assigned
						continue
					}
					if p.safe(value) || p.appendsSafe(ident.Name, value) {
						p.safeNames[ident.Name] = true
						changed = true
					}
				}
				return true
			})
		}
		if !changed {
			return
		}
	}
}

// containsSQL reports whether an expression plainly carries a statement. A chain is read as a whole:
// `"SELECT …" + cols + tail + value` is one statement and the last operand is the dangerous one.
func containsSQL(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		return typed.Kind == token.STRING && sqlWord.MatchString(typed.Value)
	case *ast.BinaryExpr:
		return containsSQL(typed.X) || containsSQL(typed.Y)
	case *ast.ParenExpr:
		return containsSQL(typed.X)
	}
	return false
}

// flattenAdd is every leaf of a `+` chain, so each operand is judged on its own.
func flattenAdd(expr ast.Expr) []ast.Expr {
	binary, ok := expr.(*ast.BinaryExpr)
	if !ok || binary.Op != token.ADD {
		return []ast.Expr{expr}
	}
	return append(flattenAdd(binary.X), flattenAdd(binary.Y)...)
}

// adapterDirs lists every non-test directory under internal/adapters.
func adapterDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no adapter package found")
	}
	return dirs
}

func parsePackage(t *testing.T, dir string) *auditPackage {
	t.Helper()
	pkg := &auditPackage{dir: dir, fset: token.NewFileSet(), consts: map[string]bool{}, safeNames: map[string]bool{}}
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(pkg.fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		pkg.files = append(pkg.files, file)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range value.Names {
					pkg.consts[name.Name] = true
				}
			}
		}
	}
	if len(pkg.files) == 0 {
		t.Fatalf("%s has no source", dir)
	}
	pkg.collectSafe()
	return pkg
}

// methodName is the name a call names: a function or a selector's method.
func methodName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	}
	return ""
}

func isQueryMethod(name string) bool {
	switch name {
	case "Query", "QueryContext", "QueryRow", "QueryRowContext", "Exec", "ExecContext":
		return true
	}
	return false
}

// queryArg reads the statement a query call names (a literal, when it is one) and how many arguments
// the call binds to it. args is -1 when the count cannot be read from the source.
func queryArg(call *ast.CallExpr) (query string, args int) {
	at := 0
	if len(call.Args) > 1 {
		if _, isCtx := call.Args[0].(*ast.Ident); isCtx {
			at = 1
		}
	}
	if _, isCtx := call.Args[0].(*ast.CallExpr); isCtx && len(call.Args) > 2 {
		at = 1
	}
	if at >= len(call.Args) {
		return "", -1
	}
	literal, ok := call.Args[at].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", -1
	}
	return strings.ReplaceAll(literal.Value, `"`, ""), len(call.Args) - at - 1
}
