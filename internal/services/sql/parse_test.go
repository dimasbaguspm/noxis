package sql

import "testing"

// splitStatements returns raw fragments; empty ones (trailing separators)
// are skipped by the callers, so counts here include them.
func TestSplitStatements(t *testing.T) {
	cases := []struct {
		sql  string
		want int
	}{
		{"SELECT 1", 1},
		{"SELECT 1; SELECT 2", 2},
		{"SELECT 1;", 2},                                // trailing separator
		{"SELECT 'a;b'", 1},                             // semicolon in a literal
		{"SELECT 'it''s; fine'", 1},                     // escaped quote
		{`SELECT "we;ird"`, 1},                          // quoted identifier
		{"SELECT 1 -- a; comment\n; SELECT 2", 2},       // line comment
		{"SELECT 1 /* a; block */; SELECT 2", 2},        // block comment
		{"INSERT INTO t VALUES ($q$it;s$q$)", 1},        // dollar-quoted
		{"INSERT INTO t VALUES ($$a;b$$); SELECT 1", 2}, // empty-tag dollars
		{"SELECT $1, $2", 1},                            // bind params stay plain
	}
	for _, tc := range cases {
		if got := len(splitStatements(tc.sql)); got != tc.want {
			t.Errorf("splitStatements(%q) = %d statements, want %d", tc.sql, got, tc.want)
		}
	}
}

func TestVerifyKindPerStatement(t *testing.T) {
	if err := verifyKind("DROP TABLE t", reads); err == nil {
		t.Error("query must reject DDL")
	}
	if err := verifyKind("SELECT 1; DROP TABLE t", reads); err == nil {
		t.Error("query must reject DDL smuggled behind a read")
	}
	if err := verifyKind("SELECT 1", reads); err != nil {
		t.Errorf("query must allow reads: %v", err)
	}
	if err := verifyKind("SELECT 1", writes); err == nil {
		t.Error("writes must reject reads")
	}
	if err := verifyKind("INSERT INTO t VALUES (1); UPDATE t SET a = 2", writes); err != nil {
		t.Errorf("writes must allow DML batches: %v", err)
	}
	if err := verifyKind("INSERT INTO t VALUES ($q$DROP TABLE x$q$)", reads); err == nil {
		t.Error("dollar-quoted text must not be read as a statement")
	}
}
