// Package sqltext is the SQL text machinery every engine needs and none of them owns: splitting a batch into statements,
// reading the first keyword, and finding what a statement touches. These are pure functions over strings, so they live
// here instead of being copied into each engine.
package sqltext

import (
	"regexp"
	"sort"
	"strings"
)

// ContainsDDL reports whether any statement is a DDL kind, per the keyword set the caller's dialect declares.
func ContainsDDL(sqlText string, ddls map[string]bool) bool {
	for _, stmt := range SplitStatements(sqlText) {
		if ddls[FirstKeyword(stmt)] {
			return true
		}
	}
	return false
}

// StatementCount reports how many non-empty statements a batch holds.
func StatementCount(sqlText string) int {
	n := 0
	for _, stmt := range SplitStatements(sqlText) {
		if StripLeadingComments(stmt) != "" {
			n++
		}
	}
	return n
}

// StripLeadingComments drops leading -- and /* */ comments, so a comment-only statement is not mistaken for real SQL.
func StripLeadingComments(s string) string {
	for {
		s = strings.TrimSpace(s)
		switch {
		case strings.HasPrefix(s, "--"):
			i := strings.IndexByte(s, '\n')
			if i < 0 {
				return ""
			}
			s = s[i+1:]
		case strings.HasPrefix(s, "/*"):
			i := strings.Index(s, "*/")
			if i < 0 {
				return ""
			}
			s = s[i+2:]
		default:
			return s
		}
	}
}

// SplitStatements splits on ';' while skipping string literals ('...' with
// ” escapes, "..." identifiers, `...` for mysql) and comments (-- and /* */),
// so semicolons inside values do not split the batch.
//
// shortcut: MySQL backslash-escaped quotes can desync splitting; use a real lexer if this syntax must be supported.
func SplitStatements(sqlText string) []string {
	var stmts []string
	var cur strings.Builder
	i, n := 0, len(sqlText)
	for i < n {
		c := sqlText[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			q := c
			cur.WriteByte(c)
			i++
			for i < n {
				cur.WriteByte(sqlText[i])
				if sqlText[i] == q {
					if i+1 < n && sqlText[i+1] == q {
						cur.WriteByte(sqlText[i+1])
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '-' && i+1 < n && sqlText[i+1] == '-':
			cur.WriteString("--")
			i += 2
			for i < n && sqlText[i] != '\n' {
				cur.WriteByte(sqlText[i])
				i++
			}
		case c == '/' && i+1 < n && sqlText[i+1] == '*':
			cur.WriteString("/*")
			i += 2
			for i < n && (sqlText[i] != '*' || i+1 >= n || sqlText[i+1] != '/') {
				cur.WriteByte(sqlText[i])
				i++
			}
			if i+1 < n {
				cur.WriteString("*/")
				i += 2
			}
		case c == '$':
			if end, ok := dollarQuoted(sqlText, i); ok {
				cur.WriteString(sqlText[i:end])
				i = end
				continue
			}
			cur.WriteByte(c)
			i++
		case c == ';':
			stmts = append(stmts, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	return append(stmts, cur.String())
}

// dollarQuoted matches a postgres dollar-quoted string ($tag$ … $tag$) at i.
func dollarQuoted(sqlText string, i int) (end int, ok bool) {
	tagEnd := strings.IndexByte(sqlText[i+1:], '$')
	if tagEnd < 0 {
		return 0, false
	}
	tag := sqlText[i : i+1+tagEnd+1] // "$tag$"
	for _, r := range tag[1 : len(tag)-1] {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return 0, false
		}
	}
	rest := sqlText[i+len(tag):]
	shut := strings.Index(rest, tag)
	if shut < 0 {
		return 0, false
	}
	return i + len(tag) + shut + len(tag), true
}

func FirstKeyword(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	if i := strings.IndexAny(trimmed, " \t\r\n"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.ToUpper(strings.Trim(trimmed, "(;"))
}

// IdentOK allows a plain or schema-qualified table name (ANALYZE and Rows interpolate identifiers; placeholders cannot be used there).
func IdentOK(s string) bool {
	if s == "" {
		return false
	}
	for _, part := range strings.Split(s, ".") {
		if part == "" {
			return false
		}
		for _, r := range part {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			default:
				return false
			}
		}
	}
	return true
}

// Qualifiers lists every identifier a dot qualifies, outside literals and comments, in the order they appear.
func Qualifiers(sqlText string) []string {
	var out []string
	WalkIdentifiers(sqlText, func(name string, qualified bool) (string, bool) {
		if qualified {
			out = append(out, name)
		}
		return "", false
	})
	return out
}

// RewriteQualifier points a qualifier that names the live database at the scratch database, so a statement the engine
// would resolve against its own database can be dry-run inside the throwaway one. Literals, quoted values and comments
// are copied untouched.
func RewriteQualifier(sqlText, from, to string) string {
	return WalkIdentifiers(sqlText, func(name string, qualified bool) (string, bool) {
		if qualified && strings.EqualFold(name, from) {
			return to, true
		}
		return "", false
	})
}

// WalkIdentifiers copies sqlText, asking visit whether to replace an identifier it finds outside string literals and
// comments. qualified reports whether a dot follows the identifier, so a schema qualifier is told apart from a table or
// column name. A backtick-quoted identifier is visited by its unescaped name.
func WalkIdentifiers(sqlText string, visit func(name string, qualified bool) (string, bool)) string {
	// replace keeps the original text unless the identifier is a qualifier and visit wants it changed.
	replace := func(name, raw string, end int) string {
		if next := skipSpace(sqlText, end); next >= len(sqlText) || sqlText[next] != '.' {
			return raw
		}
		if replacement, ok := visit(name, true); ok {
			return replacement
		}
		return raw
	}
	var b strings.Builder
	b.Grow(len(sqlText))
	for i := 0; i < len(sqlText); {
		c := sqlText[i]
		switch {
		case c == '\'':
			end := skipQuoted(sqlText, i, '\'')
			b.WriteString(sqlText[i:end])
			i = end
		case c == '`':
			end := skipQuoted(sqlText, i, '`')
			b.WriteString(replace(strings.ReplaceAll(sqlText[i+1:end-1], "``", "`"), sqlText[i:end], end))
			i = end
		case c == '-' && i+1 < len(sqlText) && sqlText[i+1] == '-':
			end := skipLine(sqlText, i)
			b.WriteString(sqlText[i:end])
			i = end
		case c == '/' && i+1 < len(sqlText) && sqlText[i+1] == '*':
			end := skipBlock(sqlText, i)
			b.WriteString(sqlText[i:end])
			i = end
		case isIdentStart(c):
			end := i + 1
			for end < len(sqlText) && isIdentByte(sqlText[end]) {
				end++
			}
			b.WriteString(replace(sqlText[i:end], sqlText[i:end], end))
			i = end
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// Keywords returns the first n words of a statement, uppercased, so a caller can recognise a statement family.
func Keywords(sqlText string, n int) []string {
	fields := strings.Fields(StripLeadingComments(sqlText))
	if len(fields) > n {
		fields = fields[:n]
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, strings.ToUpper(strings.Trim(field, "(")))
	}
	return out
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentByte(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// skipQuoted returns the offset just past the quoted run opened at i, honouring the doubled-quote escape.
//
// shortcut: a backslash-escaped quote inside a mysql string desyncs this scan the same way it desyncs SplitStatements.
func skipQuoted(sqlText string, i int, quote byte) int {
	for j := i + 1; j < len(sqlText); j++ {
		if sqlText[j] != quote {
			continue
		}
		if j+1 < len(sqlText) && sqlText[j+1] == quote {
			j++
			continue
		}
		return j + 1
	}
	return len(sqlText)
}

func skipLine(sqlText string, i int) int {
	if end := strings.IndexByte(sqlText[i:], '\n'); end >= 0 {
		return i + end
	}
	return len(sqlText)
}

func skipBlock(sqlText string, i int) int {
	if end := strings.Index(sqlText[i+2:], "*/"); end >= 0 {
		return i + 2 + end + 2
	}
	return len(sqlText)
}

func skipSpace(sqlText string, i int) int {
	for i < len(sqlText) && (sqlText[i] == ' ' || sqlText[i] == '\t' || sqlText[i] == '\r' || sqlText[i] == '\n') {
		i++
	}
	return i
}

var (
	tableRef   = regexp.MustCompile(`(?i)\b(?:from|join|update|into|table|references)\s+(?:if\s+not\s+exists\s+)?("?[\w.]+"?)`)
	insertCols = regexp.MustCompile(`(?is)\binsert\s+into\s+\S+\s*\(([^)]*)\)`)
	createCols = regexp.MustCompile(`(?is)\bcreate\s+table\s+(?:if\s+not\s+exists\s+)?\S+\s*\(([^)]*)\)`)
	createRef  = regexp.MustCompile(`(?i)\bcreate\s+(?:or\s+replace\s+)?(?:temporary\s+|temp\s+|unlogged\s+|global\s+|local\s+)?table\s+(?:if\s+not\s+exists\s+)?("?[\w.]+"?)`)
	alterCols  = regexp.MustCompile(`(?i)\b(?:add|drop|alter)\s+column\s+("?\w+"?)`)
	setCols    = regexp.MustCompile(`(?is)\bset\s+(.+?)(?:\bwhere\b|\breturning\b|$)`)
	qualified  = regexp.MustCompile(`\b[\w"]+\.("?\w+"?)`)
)

// constraint starters are not column names
var constraints = map[string]bool{
	"primary": true, "foreign": true, "unique": true, "check": true,
	"constraint": true, "key": true, "index": true, "exclude": true,
}

// Referenced extracts the tables and columns a batch mentions.
//
// shortcut: references use keyword matching, not a SQL parser; replace it when dialect coverage or precise blast radius is required.
func Referenced(sqlText string) (tables []string, columns []string) {
	tableSet := map[string]bool{}
	columnSet := map[string]bool{}

	for _, stmt := range SplitStatements(sqlText) {
		if FirstKeyword(stmt) == "" {
			continue
		}
		for _, match := range tableRef.FindAllStringSubmatch(stmt, -1) {
			if name := cleanIdent(match[1]); name != "" {
				tableSet[name] = true
			}
		}
		for _, match := range insertCols.FindAllStringSubmatch(stmt, -1) {
			addList(columnSet, match[1], false)
		}
		for _, match := range createCols.FindAllStringSubmatch(stmt, -1) {
			addList(columnSet, match[1], true)
		}
		for _, match := range alterCols.FindAllStringSubmatch(stmt, -1) {
			if name := cleanIdent(match[1]); name != "" {
				columnSet[name] = true
			}
		}
		for _, match := range setCols.FindAllStringSubmatch(stmt, -1) {
			for _, assignment := range strings.Split(match[1], ",") {
				left, _, found := strings.Cut(assignment, "=")
				if !found {
					continue
				}
				if name := cleanIdent(left); name != "" {
					columnSet[name] = true
				}
			}
		}
		// qualified references (t.col) — with the table targets removed so "ALTER TABLE public.items" does not register "items" as a column
		body := tableRef.ReplaceAllString(stmt, " _target_ ")
		for _, match := range qualified.FindAllStringSubmatch(body, -1) {
			if name := cleanIdent(match[1]); name != "" {
				columnSet[name] = true
			}
		}
	}

	return sorted(tableSet), sorted(columnSet)
}

// CreatedTables finds tables the batch creates so scratch verification does not mirror them first.
func CreatedTables(sqlText string) map[string]bool {
	set := map[string]bool{}
	for _, stmt := range SplitStatements(sqlText) {
		for _, match := range createRef.FindAllStringSubmatch(stmt, -1) {
			if name := cleanIdent(match[1]); name != "" {
				set[name] = true
			}
		}
	}
	return set
}

// addList records the identifiers of a parenthesised list; for CREATE TABLE bodies each item's first word is the column unless it starts a constraint.
func addList(set map[string]bool, list string, firstOnly bool) {
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		word := item
		if firstOnly {
			word = strings.Fields(item)[0]
		}
		name := cleanIdent(word)
		if name == "" || constraints[strings.ToLower(name)] {
			continue
		}
		set[name] = true
	}
}

func cleanIdent(raw string) string {
	name := strings.Trim(strings.TrimSpace(raw), `"`)
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t\r\n,()") {
		return ""
	}
	return name
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
