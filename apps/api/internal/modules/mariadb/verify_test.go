package mariadb

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"noxis/internal/entity"
)

func TestCreatedTables(t *testing.T) {
	cases := []struct {
		sql  string
		want []string
	}{
		{"CREATE TABLE notes (id INT)", []string{"notes"}},
		{"CREATE TABLE IF NOT EXISTS public.notes (id INT)", []string{"public.notes"}},
		{"CREATE OR REPLACE TABLE t (id INT)", []string{"t"}},
		{"CREATE TEMPORARY TABLE tmp (id INT)", []string{"tmp"}},
		{"CREATE INDEX idx ON notes (id)", nil},
		{"ALTER TABLE notes ADD COLUMN tag TEXT", nil},
		{"CREATE TABLE a (id INT); CREATE TABLE b (id INT)", []string{"a", "b"}},
	}
	for _, tc := range cases {
		got := mapKeys(createdTables(tc.sql))
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: createdTables = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

func TestRewriteQualifier(t *testing.T) {
	cases := []struct {
		sql  string
		want string
	}{
		{"ALTER TABLE demo.orders ADD COLUMN x INT", "ALTER TABLE noxis_verify_x.orders ADD COLUMN x INT"},
		{"ALTER TABLE `demo`.`orders` ADD COLUMN x INT", "ALTER TABLE noxis_verify_x.`orders` ADD COLUMN x INT"},
		{"ALTER TABLE demo . orders ADD COLUMN x INT", "ALTER TABLE noxis_verify_x . orders ADD COLUMN x INT"},
		{"ALTER TABLE other.orders ADD COLUMN x INT", "ALTER TABLE other.orders ADD COLUMN x INT"},
		{"INSERT INTO demo.notes (body) VALUES ('demo.notes')", "INSERT INTO noxis_verify_x.notes (body) VALUES ('demo.notes')"},
		{"ALTER TABLE demo.orders ADD COLUMN x INT -- demo.orders stays", "ALTER TABLE noxis_verify_x.orders ADD COLUMN x INT -- demo.orders stays"},
		{"SELECT 1.5 FROM demo.orders", "SELECT 1.5 FROM noxis_verify_x.orders"},
		{"SELECT * FROM `demo` WHERE note = 'demo.orders'", "SELECT * FROM `demo` WHERE note = 'demo.orders'"},
	}
	for _, tc := range cases {
		if got := rewriteQualifier(tc.sql, "demo", "noxis_verify_x"); got != tc.want {
			t.Errorf("rewriteQualifier(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

func TestUnprovableReason(t *testing.T) {
	conn := entity.Connection{Driver: "mysql", Database: "demo"}
	for _, sql := range []string{
		"ALTER TABLE other.orders ADD COLUMN x INT",
		"CREATE DATABASE other",
		"GRANT ALL ON demo.* TO 'x'@'%'",
		"CREATE USER 'x'@'%'",
		"SET GLOBAL max_connections = 100",
	} {
		if reason := unprovableReason(conn, sql); reason == "" {
			t.Errorf("unprovableReason(%q) = \"\", want a reason", sql)
		}
	}
	for _, sql := range []string{
		"ALTER TABLE demo.orders ADD COLUMN x INT",
		"ALTER TABLE orders ADD COLUMN x INT",
		"CREATE TABLE demo.notes (id INT)",
		"SELECT COUNT(*) FROM information_schema.tables",
	} {
		if reason := unprovableReason(conn, sql); reason != "" {
			t.Errorf("unprovableReason(%q) = %q, want no reason", sql, reason)
		}
	}
}

func TestScratchSchemaName(t *testing.T) {
	name := scratchPrefix + "0123456789ab"
	if !strings.HasPrefix(name, "noxis_verify_") {
		t.Fatalf("scratch name %q lacks the prefix", name)
	}
	if len(name) > 63 {
		t.Fatalf("scratch name %q exceeds the postgres identifier limit", name)
	}
}

func mapKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
