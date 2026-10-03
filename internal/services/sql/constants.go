package sql

import "strings"

// Statement kind guards: query is read-only; alter takes every write (DML and
// DDL). WITH is a write opener (CTEs can front INSERT/UPDATE/DELETE).
const (
	readKinds  = "SELECT SHOW EXPLAIN DESCRIBE DESC"
	dmlKinds   = "WITH INSERT UPDATE DELETE MERGE REPLACE"
	ddlKinds   = "ALTER CREATE DROP TRUNCATE RENAME GRANT REVOKE COMMENT"
	writeKinds = dmlKinds + " " + ddlKinds
)

const (
	defaultPreviewRows = 100
	maxPreviewRows     = 500
)

// kindSet is an allow-list of statement keywords, with its printable list.
type kindSet struct {
	names string
	set   map[string]bool
}

func newKindSet(names string) kindSet {
	set := map[string]bool{}
	for _, k := range strings.Fields(names) {
		set[k] = true
	}
	return kindSet{names: names, set: set}
}

var (
	reads  = newKindSet(readKinds)
	writes = newKindSet(writeKinds)
	ddls   = newKindSet(ddlKinds)
)
