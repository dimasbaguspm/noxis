package sql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"noxis/internal/entity"
)

// verifyKind rejects every statement whose first keyword is not allowed. The
// check runs per statement — split outside quoted strings and comments — so
// DDL cannot be smuggled behind a read (`SELECT 1; DROP TABLE x`).
func verifyKind(sqlText string, allowed kindSet) error {
	for _, stmt := range splitStatements(sqlText) {
		if kind := firstKeyword(stmt); kind != "" && !allowed.set[kind] {
			return fmt.Errorf("statement %q not allowed here (only: %s)", kind, allowed.names)
		}
	}
	return nil
}

// ErrUnverifiable reports that a change cannot be dry-run on this driver
// (mysql commits DDL implicitly), so validation is skipped honestly.
var ErrUnverifiable = errors.New("unverifiable on this driver")

// IsUnverifiable reports the driver-cannot-dry-run case.
func IsUnverifiable(err error) bool { return errors.Is(err, ErrUnverifiable) }

// containsDDL reports whether any statement is a DDL kind.
func containsDDL(sqlText string) bool {
	for _, stmt := range splitStatements(sqlText) {
		if ddls.set[firstKeyword(stmt)] {
			return true
		}
	}
	return false
}

// splitStatements splits on ';' while skipping string literals ('...' with
// ” escapes, "..." identifiers, `...` for mysql) and comments (-- and /* */),
// so semicolons inside values do not split the batch.
//
// ponytail: backslash-escaped quotes in literals (mysql without
// NO_BACKSLASH_ESCAPES) can still desync the splitter — a real lexer if that
// ever matters.
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

func firstKeyword(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	if i := strings.IndexAny(trimmed, " \t\r\n"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.ToUpper(strings.Trim(trimmed, "(;"))
}

// identOK allows a plain or schema-qualified table name (ANALYZE and Rows
// interpolate identifiers; placeholders cannot be used there).
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
	if isMySQL(c.Driver) {
		if schema != "" {
			return "`" + schema + "`.`" + table + "`"
		}
		return "`" + table + "`"
	}
	if schema != "" {
		return `"` + schema + `"."` + table + `"`
	}
	return `"` + table + `"`
}

func itoa(n int) string { return strconv.Itoa(n) }
