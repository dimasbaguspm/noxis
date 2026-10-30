package sqlite

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestProfileIsComplete fails when the profile leaves out a field a feature reads: the compiler cannot see the gap, this test can.
func TestProfileIsComplete(t *testing.T) {
	d := profile
	if d.name != constants.DriverSQLite || d.driver != dbDriverSQLite || d.schema != schemaSQLite || d.quote != '"' {
		t.Fatalf("profile identity = %+v", d)
	}
	if d.dsn == nil || d.analyze == "" || d.maxOpenConns <= 0 || d.txn != txnAlways {
		t.Fatalf("profile is missing a required field: %+v", d)
	}
	// no information_schema: every table is read through pragmas, so there are no catalog queries to carry
	if !d.catalogPerTable || d.keysQuery != "" || d.indexesQuery != "" {
		t.Fatalf("profile = %+v, want the per-table catalog", d)
	}
	// EXPLAIN QUERY PLAN reports access paths, not row counts
	if d.explain != "" || d.estimatesQuery != "" {
		t.Fatalf("this engine reports no planner estimate: %+v", d)
	}
	if len(d.capabilities()) == 0 {
		t.Fatal("capabilities must say what the profile proves")
	}
	if !d.transactional("CREATE TABLE t (id INT)") || !d.transactional("UPDATE public.t SET id = 1") {
		t.Fatal("every batch is proved by rolling back the live transaction")
	}
}

// TestModuleServesOneDriver proves the package answers for its own driver and refuses every other.
func TestModuleServesOneDriver(t *testing.T) {
	db := New()
	if db.Driver() != constants.DriverSQLite {
		t.Fatalf("Driver() = %q", db.Driver())
	}
	if !db.Supports(constants.DriverSQLite) || !db.Supports("SQLITE") {
		t.Fatal("Supports must accept its own driver")
	}
	for _, other := range []string{constants.DriverPostgres, constants.DriverMySQL, constants.DriverMariaDB, constants.DriverCockroach, "oracle", ""} {
		if db.Supports(other) {
			t.Errorf("Supports(%q) = true", other)
		}
	}
	if _, err := dialectOf(constants.DriverPostgres); err == nil {
		t.Error("dialectOf must refuse a foreign driver")
	}
	if got, err := dialectOf(constants.DriverSQLite); err != nil || got.driver != dbDriverSQLite {
		t.Errorf("dialectOf(sqlite) = %+v, %v", got, err)
	}
}

// TestDSNAndSplitRef pins how a connection becomes a wire DSN and where a bare table name resolves.
func TestDSNAndSplitRef(t *testing.T) {
	conn := entity.Connection{Driver: constants.DriverSQLite, Database: "/tmp/app.db"}
	got, err := dsn(conn)
	if err != nil || got != "/tmp/app.db" {
		t.Fatalf("dsn = %q, %v", got, err)
	}
	withParams := conn
	withParams.Params = map[string]string{"_pragma": "busy_timeout(5000)"}
	if got, err := dsn(withParams); err != nil || got != "file:/tmp/app.db?_pragma=busy_timeout%285000%29" {
		t.Fatalf("dsn with params = %q, %v", got, err)
	}
	if schema, table := splitRef(conn, "items"); schema != schemaSQLite || table != "items" {
		t.Fatalf("splitRef(items) = %q.%q", schema, table)
	}
	if quoted := quoteRef(conn.Driver, "s", "t"); !strings.EqualFold(quoted, `"s"."t"`) {
		t.Fatalf("quoteRef = %q", quoted)
	}
}
