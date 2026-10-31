package mysql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"noxis/internal/common/sqltext"
	"noxis/internal/entity"
)

// verifyKind checks every statement independently so writes cannot be hidden behind a read.
func verifyKind(sqlText string, allowed kindSet) error {
	for _, stmt := range sqltext.SplitStatements(sqlText) {
		if kind := sqltext.FirstKeyword(stmt); kind != "" && !allowed.set[kind] {
			return fmt.Errorf("statement %q not allowed here (only: %s)", kind, allowed.names)
		}
	}
	return nil
}

// StatementCount reports how many non-empty statements a batch holds.
func StatementCount(sqlText string) int {
	n := 0
	for _, stmt := range sqltext.SplitStatements(sqlText) {
		if sqltext.StripLeadingComments(stmt) != "" {
			n++
		}
	}
	return n
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

// qualify quotes a table reference per driver (identifiers were validated).
func qualify(c entity.Connection, schema, table string) string {
	return quoteRef(c.Driver, schema, table)
}

func itoa(n int) string { return strconv.Itoa(n) }
