package sql

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"noxis/internal/constants"
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

func TestSplitRef(t *testing.T) {
	pg := entity.Connection{Driver: constants.DriverPostgres, Database: "demo"}
	my := entity.Connection{Driver: constants.DriverMySQL, Database: "demo"}
	cases := []struct {
		conn       entity.Connection
		ref        string
		wantSchema string
		wantTable  string
	}{
		{pg, "items", "public", "items"},
		{pg, "public.items", "public", "items"},
		{my, "items", "demo", "items"},
		{my, "demo.items", "demo", "items"},
	}
	for _, tc := range cases {
		schema, table := splitRef(tc.conn, tc.ref)
		if schema != tc.wantSchema || table != tc.wantTable {
			t.Errorf("splitRef(%q) = %q.%q, want %q.%q", tc.ref, schema, table, tc.wantSchema, tc.wantTable)
		}
	}
}

func TestQualifiedIdent(t *testing.T) {
	for _, sql := range []string{
		"DROP TABLE public.victim",
		"UPDATE t SET t.name = 'x'",
		"ALTER TABLE `demo`.`items` ADD COLUMN x INT",
	} {
		if !hasQualifiedIdent(sql) {
			t.Errorf("hasQualifiedIdent(%q) = false", sql)
		}
	}
	for _, sql := range []string{
		"DROP TABLE victim",
		"UPDATE items SET name = 'x'",
		"SELECT 1",
	} {
		if hasQualifiedIdent(sql) {
			t.Errorf("hasQualifiedIdent(%q) = true", sql)
		}
	}
}

func TestTransactionSafe(t *testing.T) {
	if !transactionSafe(constants.DriverPostgres, "DROP TABLE public.victim") {
		t.Fatal("qualified postgres statement must use a live transaction")
	}
	if !transactionSafe(constants.DriverMySQL, "UPDATE items SET name = 'x'") {
		t.Fatal("mysql DML can use a live transaction")
	}
	if transactionSafe(constants.DriverMySQL, "DROP TABLE demo.victim") {
		t.Fatal("qualified mysql DDL cannot use a live transaction")
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
