package mysql

import (
	"regexp"
	"sort"
	"strings"
)

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

	for _, stmt := range splitStatements(sqlText) {
		if firstKeyword(stmt) == "" {
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

// createdTables finds tables the batch creates so scratch verification does not mirror them first.
func createdTables(sqlText string) map[string]bool {
	set := map[string]bool{}
	for _, stmt := range splitStatements(sqlText) {
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
