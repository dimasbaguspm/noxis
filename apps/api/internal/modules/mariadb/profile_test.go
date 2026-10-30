package mariadb

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestProfileIsComplete fails when the profile leaves out a field a feature reads: the compiler cannot see the gap, this test can.
func TestProfileIsComplete(t *testing.T) {
	d := profile
	// the profile names the engine "mysql": the same protocol and catalog, and the console has always reported it that way.
	if d.name != constants.DriverMySQL || d.driver != dbDriverMySQL || d.schema != "" || d.quote != '`' {
		t.Fatalf("profile identity = %+v", d)
	}
	if d.dsn == nil || d.analyze == "" || d.maxOpenConns <= 0 || d.txn != txnWithoutDDL {
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
	// DDL commits implicitly here, so only DML can be proved in the live transaction
	if !d.transactional("UPDATE t SET id = 1") || d.transactional("CREATE TABLE t (id INT)") {
		t.Fatal("only DML may be proved in the live transaction")
	}
	if !d.qualifiedDDLRefused("CREATE TABLE demo.t (id INT)") || !d.refuseQualifiedDDL {
		t.Fatal("qualified DDL cannot be dry-run in a scratch schema")
	}
}

// TestModuleServesOneDriver proves the package answers for its own driver and refuses every other.
func TestModuleServesOneDriver(t *testing.T) {
	db := New()
	if db.Driver() != constants.DriverMariaDB {
		t.Fatalf("Driver() = %q", db.Driver())
	}
	if !db.Supports(constants.DriverMariaDB) || !db.Supports("MARIADB") {
		t.Fatal("Supports must accept its own driver")
	}
	for _, other := range []string{constants.DriverPostgres, constants.DriverMySQL, constants.DriverSQLite, constants.DriverCockroach, "oracle", ""} {
		if db.Supports(other) {
			t.Errorf("Supports(%q) = true", other)
		}
	}
	if _, err := dialectOf(constants.DriverMySQL); err == nil {
		t.Error("dialectOf must refuse a foreign driver")
	}
	if got, err := dialectOf(constants.DriverMariaDB); err != nil || got.driver != dbDriverMySQL {
		t.Errorf("dialectOf(mariadb) = %+v, %v", got, err)
	}
}

// TestDSNAndSplitRef pins how a connection becomes a wire DSN and where a bare table name resolves.
func TestDSNAndSplitRef(t *testing.T) {
	conn := entity.Connection{Driver: constants.DriverMariaDB, Host: "db", Port: 3306, User: "u", Password: "p", Database: "demo"}
	got, err := dsn(conn)
	if err != nil || !strings.Contains(got, "tcp(db:3306)/demo") {
		t.Fatalf("dsn = %q, %v", got, err)
	}
	if schema, table := splitRef(conn, "items"); schema != "demo" || table != "items" {
		t.Fatalf("splitRef(items) = %q.%q", schema, table)
	}
	if quoted := quoteRef(conn.Driver, "s", "t"); quoted != "`s`.`t`" {
		t.Fatalf("quoteRef = %q", quoted)
	}
}
