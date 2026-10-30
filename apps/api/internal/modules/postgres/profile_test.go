package postgres

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestProfileIsComplete fails when the profile leaves out a field a feature reads: the compiler cannot see the gap, this test can.
func TestProfileIsComplete(t *testing.T) {
	d := profile
	if d.name != constants.DriverPostgres || d.driver != dbDriverPostgres || d.schema != schemaPostgres || d.quote != '"' {
		t.Fatalf("profile identity = %+v", d)
	}
	if d.dsn == nil || d.analyze == "" || d.maxOpenConns <= 0 || d.txn != txnWhenQualified {
		t.Fatalf("profile is missing a required field: %+v", d)
	}
	if d.keysQuery == "" || d.indexesQuery == "" || d.scanIndex == nil || d.scanEstimate == nil {
		t.Fatalf("the catalog the browser depends on is missing: %+v", d)
	}
	if d.estimatesQuery == "" || d.commentsQuery == "" || d.explain == "" {
		t.Fatalf("this engine reports row estimates and table comments: %+v", d)
	}
	if len(d.capabilities()) == 0 {
		t.Fatal("capabilities must say what the profile proves")
	}
	if d.transactional("UPDATE t SET id = 1") || !d.transactional("UPDATE public.t SET id = 1") {
		t.Fatal("a qualified name must fall back to the live transaction")
	}
}

// TestModuleServesOneDriver proves the package answers for its own driver and refuses every other.
func TestModuleServesOneDriver(t *testing.T) {
	db := New()
	if db.Driver() != constants.DriverPostgres {
		t.Fatalf("Driver() = %q", db.Driver())
	}
	if !db.Supports(constants.DriverPostgres) || !db.Supports("POSTGRES") {
		t.Fatal("Supports must accept its own driver")
	}
	for _, other := range []string{constants.DriverMySQL, constants.DriverMariaDB, constants.DriverSQLite, constants.DriverCockroach, "oracle", ""} {
		if db.Supports(other) {
			t.Errorf("Supports(%q) = true", other)
		}
	}
	if _, err := dialectOf(constants.DriverMySQL); err == nil {
		t.Error("dialectOf must refuse a foreign driver")
	}
	if got, err := dialectOf(constants.DriverPostgres); err != nil || got.name != constants.DriverPostgres {
		t.Errorf("dialectOf(postgres) = %+v, %v", got, err)
	}
	// the long form of the same engine resolved before the split and still does
	if got, err := dialectOf("postgresql"); err != nil || got.driver != dbDriverPostgres {
		t.Errorf("dialectOf(postgresql) = %+v, %v", got, err)
	}
}

// TestDSNAndSplitRef pins how a connection becomes a wire DSN and where a bare table name resolves.
func TestDSNAndSplitRef(t *testing.T) {
	conn := entity.Connection{Driver: constants.DriverPostgres, Host: "db", Port: 5432, User: "u", Password: "p", Database: "demo"}
	got, err := dsn(conn)
	if err != nil || !strings.HasPrefix(got, "postgres://u:p@db:5432/demo") {
		t.Fatalf("dsn = %q, %v", got, err)
	}
	if schema, table := splitRef(conn, "items"); schema != schemaPostgres || table != "items" {
		t.Fatalf("splitRef(items) = %q.%q", schema, table)
	}
	if schema, table := splitRef(conn, "public.items"); schema != "public" || table != "items" {
		t.Fatalf("splitRef(public.items) = %q.%q", schema, table)
	}
	if quoted := quoteRef(conn.Driver, "s", "t"); quoted != `"s"."t"` {
		t.Fatalf("quoteRef = %q", quoted)
	}
}
