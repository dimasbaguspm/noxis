package postgres

import (
	"reflect"
	"testing"
)

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
		tables, columns := Referenced(tc.sql)
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
	tables, _ := Referenced(`CREATE TABLE child (id INT PRIMARY KEY, parent_id INT REFERENCES parent (id))`)
	if len(tables) != 2 || tables[0] != "child" || tables[1] != "parent" {
		t.Fatalf("tables = %v, want [child parent]", tables)
	}
}
