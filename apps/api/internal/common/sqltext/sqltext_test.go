package sqltext_test

import (
	"reflect"
	"testing"

	"noxis/internal/common/sqltext"
)

// splitStatements returns raw fragments; empty ones (trailing separators) are skipped by the callers, so counts here include them.
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
		if got := len(sqltext.SplitStatements(tc.sql)); got != tc.want {
			t.Errorf("sqltext.SplitStatements(%q) = %d statements, want %d", tc.sql, got, tc.want)
		}
	}
}

func TestReferenced(t *testing.T) {
	cases := []struct {
		sql     string
		tables  []string
		columns []string
	}{
		{
			"INSERT INTO items (id, name) VALUES (1, 'x')",
			[]string{"items"},
			[]string{"id", "name"},
		},
		{
			"UPDATE items SET name = 'y', updated_at = now() WHERE id = 1",
			[]string{"items"},
			[]string{"name", "updated_at"},
		},
		{
			"ALTER TABLE public.items ADD COLUMN tag TEXT",
			[]string{"public.items"},
			[]string{"tag"},
		},
		{
			"CREATE TABLE IF NOT EXISTS notes (id INT, body TEXT, created_at TIMESTAMP, PRIMARY KEY (id))",
			[]string{"notes"},
			[]string{"body", "created_at", "id"},
		},
		{
			"SELECT * FROM orders o JOIN customers c ON o.customer_id = c.id",
			[]string{"customers", "orders"},
			[]string{"customer_id", "id"},
		},
		{"DELETE FROM items WHERE id = 1; DROP TABLE old_items", []string{"items", "old_items"}, []string{}},
	}
	for _, tc := range cases {
		tables, columns := sqltext.Referenced(tc.sql)
		if !reflect.DeepEqual(tables, tc.tables) {
			t.Errorf("%s\ntables = %v, want %v", tc.sql, tables, tc.tables)
		}
		if !reflect.DeepEqual(columns, tc.columns) {
			t.Errorf("%s\ncolumns = %v, want %v", tc.sql, columns, tc.columns)
		}
	}
}

// MySQL resolves unqualified names only in scratch, so the verifier must mirror FK-only parent tables.
func TestReferencedIncludesForeignKeyTargets(t *testing.T) {
	tables, _ := sqltext.Referenced(`CREATE TABLE child (id INT PRIMARY KEY, parent_id INT REFERENCES parent (id))`)
	if len(tables) != 2 || tables[0] != "child" || tables[1] != "parent" {
		t.Fatalf("tables = %v, want [child parent]", tables)
	}
}
