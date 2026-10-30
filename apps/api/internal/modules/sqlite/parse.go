package sqlite

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
