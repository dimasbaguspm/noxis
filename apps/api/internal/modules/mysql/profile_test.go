package mysql

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestProfileIsComplete fails when the profile leaves out a field a feature reads: the compiler cannot see the gap, this test can.
func TestProfileIsComplete(t *testing.T) {
	d := profile
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
	// A qualifier naming the connection's own database is rewritten into the scratch database, so it is provable; a
	// qualifier naming a different database is not, and says so.
	conn := entity.Connection{Driver: d.name, Database: "demo"}
	if err := unprovable(conn, "CREATE TABLE demo.t (id INT)"); err != nil {
		t.Fatalf("a same-database qualifier must be provable: %s", err)
	}
	if err := unprovable(conn, "CREATE TABLE other.t (id INT)"); err == nil {
		t.Fatal("a qualifier in another database cannot be dry-run in a scratch schema")
	}
}

// TestModuleServesOneDriver proves the package answers for its own driver and refuses every other.
func TestModuleServesOneDriver(t *testing.T) {
	db := New()
	if db.Driver() != constants.DriverMySQL {
		t.Fatalf("Driver() = %q", db.Driver())
	}
	if !db.Supports(constants.DriverMySQL) || !db.Supports("MYSQL") {
		t.Fatal("Supports must accept its own driver")
	}
	for _, other := range []string{constants.DriverPostgres, constants.DriverMariaDB, constants.DriverSQLite, constants.DriverCockroach, "oracle", ""} {
		if db.Supports(other) {
			t.Errorf("Supports(%q) = true", other)
		}
	}
	if _, err := dialectOf(constants.DriverMariaDB); err == nil {
		t.Error("dialectOf must refuse a foreign driver")
	}
	if got, err := dialectOf(constants.DriverMySQL); err != nil || got.driver != dbDriverMySQL {
		t.Errorf("dialectOf(mysql) = %+v, %v", got, err)
	}
}

// TestDSNAndSplitRef pins how a connection becomes a wire DSN and where a bare table name resolves.
func TestDSNAndSplitRef(t *testing.T) {
	conn := entity.Connection{Driver: constants.DriverMySQL, Host: "db", Port: 3306, User: "u", Password: "p", Database: "demo"}
	got, err := dsn(conn)
	if err != nil || !strings.Contains(got, "tcp(db:3306)/demo") {
		t.Fatalf("dsn = %q, %v", got, err)
	}
	// a database-per-schema engine resolves a bare name to the connection's own database
	if schema, table := splitRef(conn, "items"); schema != "demo" || table != "items" {
		t.Fatalf("splitRef(items) = %q.%q", schema, table)
	}
	if schema, table := splitRef(conn, "demo.items"); schema != "demo" || table != "items" {
		t.Fatalf("splitRef(demo.items) = %q.%q", schema, table)
	}
	if quoted := quoteRef(conn.Driver, "s", "t"); quoted != "`s`.`t`" {
		t.Fatalf("quoteRef = %q", quoted)
	}
}
