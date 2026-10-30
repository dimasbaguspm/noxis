package cockroach

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestProfileIsComplete fails when the profile leaves out a field a feature reads: the compiler cannot see the gap, this test can.
func TestProfileIsComplete(t *testing.T) {
	d := profile
	if d.name != constants.DriverCockroach || d.driver != dbDriverPostgres || d.schema != schemaPostgres || d.quote != '"' {
		t.Fatalf("profile identity = %+v", d)
	}
	if d.dsn == nil || d.analyze == "" || d.maxOpenConns <= 0 || d.txn != txnAlways {
		t.Fatalf("profile is missing a required field: %+v", d)
	}
	if d.keysQuery == "" || d.indexesQuery == "" || d.scanIndex == nil || d.scanEstimate == nil || d.commentsQuery == "" {
		t.Fatalf("the postgres catalog must survive the derivation: %+v", d)
	}
	// no pg_total_relation_size and no planner row counts, so both stay empty
	if d.estimatesQuery != "" || d.explain != "" {
		t.Fatalf("this engine reports no size and no estimate: %+v", d)
	}
	if len(d.capabilities()) == 0 {
		t.Fatal("capabilities must say what the profile proves")
	}
	if !d.transactional("CREATE TABLE t (id INT)") || !d.transactional("UPDATE public.t SET id = 1") {
		t.Fatal("every batch is proved by rolling back the live transaction")
	}
	// the derivation is visible: the baseline it copies is the postgres profile in this package
	if postgresBaseline.name != constants.DriverPostgres || postgresBaseline.txn != txnWhenQualified {
		t.Fatalf("postgresBaseline = %+v", postgresBaseline)
	}
}

// TestModuleServesOneDriver proves the package answers for its own driver and refuses every other.
func TestModuleServesOneDriver(t *testing.T) {
	db := New()
	if db.Driver() != constants.DriverCockroach {
		t.Fatalf("Driver() = %q", db.Driver())
	}
	if !db.Supports(constants.DriverCockroach) || !db.Supports("COCKROACH") {
		t.Fatal("Supports must accept its own driver")
	}
	for _, other := range []string{constants.DriverPostgres, constants.DriverMySQL, constants.DriverMariaDB, constants.DriverSQLite, "oracle", ""} {
		if db.Supports(other) {
			t.Errorf("Supports(%q) = true", other)
		}
	}
	// the postgres driver name must not reach this module: it is the baseline, not the driver served
	if _, err := dialectOf(constants.DriverPostgres); err == nil {
		t.Error("dialectOf must refuse a foreign driver")
	}
	if got, err := dialectOf(constants.DriverCockroach); err != nil || got.name != constants.DriverCockroach {
		t.Errorf("dialectOf(cockroach) = %+v, %v", got, err)
	}
}

// TestDSNAndSplitRef pins how a connection becomes a wire DSN and where a bare table name resolves.
func TestDSNAndSplitRef(t *testing.T) {
	conn := entity.Connection{Driver: constants.DriverCockroach, Host: "db", Port: 26257, User: "u", Password: "p", Database: "demo"}
	got, err := dsn(conn)
	if err != nil || !strings.HasPrefix(got, "postgres://u:p@db:26257/demo") {
		t.Fatalf("dsn = %q, %v", got, err)
	}
	if schema, table := splitRef(conn, "items"); schema != schemaPostgres || table != "items" {
		t.Fatalf("splitRef(items) = %q.%q", schema, table)
	}
	if quoted := quoteRef(conn.Driver, "s", "t"); quoted != `"s"."t"` {
		t.Fatalf("quoteRef = %q", quoted)
	}
}
