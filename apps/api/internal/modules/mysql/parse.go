package mysql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"noxis/internal/entity"
)

// verifyKind checks every statement independently so writes cannot be hidden behind a read.
func verifyKind(sqlText string, allowed kindSet) error {
	for _, stmt := range splitStatements(sqlText) {
		if kind := firstKeyword(stmt); kind != "" && !allowed.set[kind] {
			return fmt.Errorf("statement %q not allowed here (only: %s)", kind, allowed.names)
		}
	}
	return nil
}

// containsDDL reports whether any statement is a DDL kind.
func containsDDL(sqlText string) bool {
	for _, stmt := range splitStatements(sqlText) {
		if ddls.set[firstKeyword(stmt)] {
			return true
		}
	}
	return false
}

// StatementCount reports how many non-empty statements a batch holds.
func StatementCount(sqlText string) int {
	n := 0
	for _, stmt := range splitStatements(sqlText) {
		if stripLeadingComments(stmt) != "" {
			n++
		}
	}
	return n
}

// stripLeadingComments drops leading -- and /* */ comments, so a comment-only statement is not mistaken for real SQL.
func stripLeadingComments(s string) string {
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

// splitStatements splits on ';' while skipping string literals ('...' with
// ” escapes, "..." identifiers, `...` for mysql) and comments (-- and /* */),
// so semicolons inside values do not split the batch.
//
// shortcut: MySQL backslash-escaped quotes can desync splitting; use a real lexer if this syntax must be supported.
func splitStatements(sqlText string) []string {
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

// hintOf augments common engine errors with the fix, so a rejected change teaches instead of just refusing.
var quotedValue = regexp.MustCompile(`(?i)(?:values\s*\(|=\s*)"([^"]+)"`)

func hintOf(sqlText string, err error) error {
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		return err
	}
	if match := quotedValue.FindStringSubmatch(sqlText); match != nil {
		return fmt.Errorf(`%w — hint: "%s" is an identifier in postgres; use '%s' for a string literal`, err, match[1], match[1])
	}
	return err
}

func firstKeyword(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	if i := strings.IndexAny(trimmed, " \t\r\n"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.ToUpper(strings.Trim(trimmed, "(;"))
}

// identOK allows a plain or schema-qualified table name (ANALYZE and Rows interpolate identifiers; placeholders cannot be used there).
func identOK(s string) bool {
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

// qualify quotes a table reference per driver (identifiers were validated).
func qualify(c entity.Connection, schema, table string) string {
	return quoteRef(c.Driver, schema, table)
}

func itoa(n int) string { return strconv.Itoa(n) }

// qualifiers lists every identifier a dot qualifies, outside literals and comments, in the order they appear.
func qualifiers(sqlText string) []string {
	var out []string
	walkIdentifiers(sqlText, func(name string, qualified bool) (string, bool) {
		if qualified {
			out = append(out, name)
		}
		return "", false
	})
	return out
}

// rewriteQualifier points a qualifier that names the live database at the scratch database, so a statement the engine
// would resolve against its own database can be dry-run inside the throwaway one. Literals, quoted values and comments
// are copied untouched.
func rewriteQualifier(sqlText, from, to string) string {
	return walkIdentifiers(sqlText, func(name string, qualified bool) (string, bool) {
		if qualified && strings.EqualFold(name, from) {
			return to, true
		}
		return "", false
	})
}

// walkIdentifiers copies sqlText, asking visit whether to replace an identifier it finds outside string literals and
// comments. qualified reports whether a dot follows the identifier, so a schema qualifier is told apart from a table or
// column name. A backtick-quoted identifier is visited by its unescaped name.
func walkIdentifiers(sqlText string, visit func(name string, qualified bool) (string, bool)) string {
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

// keywords returns the first n words of a statement, uppercased, so a caller can recognise a statement family.
func keywords(sqlText string, n int) []string {
	fields := strings.Fields(stripLeadingComments(sqlText))
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
// shortcut: a backslash-escaped quote inside a mysql string desyncs this scan the same way it desyncs splitStatements.
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
