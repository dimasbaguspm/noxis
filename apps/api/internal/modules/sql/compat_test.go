package sql

import (
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// The four supported SQL drivers and the dialect decisions that differ between them.
func TestDriverCompatibility(t *testing.T) {
	cases := []struct {
		driver       string
		wantDBDriver string
		mysqlFamily  bool
		fileBacked   bool
		wantSchema   string
	}{
		{constants.DriverPostgres, dbDriverPostgres, false, false, schemaPostgres},
		{constants.DriverMySQL, dbDriverMySQL, true, false, "demo"},
		{constants.DriverMariaDB, dbDriverMySQL, true, false, "demo"},
		{constants.DriverSQLite, dbDriverSQLite, false, true, schemaSQLite},
	}
	for _, tc := range cases {
		if got, err := sqlDriver(tc.driver); err != nil || got != tc.wantDBDriver {
			t.Errorf("%s: sqlDriver = %q, %v; want %q", tc.driver, got, err, tc.wantDBDriver)
		}
		if got := isMySQL(tc.driver); got != tc.mysqlFamily {
			t.Errorf("%s: isMySQL = %v", tc.driver, got)
		}
		if got := isSQLite(tc.driver); got != tc.fileBacked {
			t.Errorf("%s: isSQLite = %v", tc.driver, got)
		}
		// a bare name resolves to the connection's own schema
		schema, table := splitRef(entity.Connection{Driver: tc.driver, Database: "demo"}, "t")
		if schema != tc.wantSchema || table != "t" {
			t.Errorf("%s: splitRef = %q.%q, want %q.t", tc.driver, schema, table, tc.wantSchema)
		}
		quoted := quoteRef(tc.driver, "s", "t")
		if tc.mysqlFamily && quoted != "`s`.`t`" {
			t.Errorf("%s: quoteRef = %q, want backticks", tc.driver, quoted)
		}
		if !tc.mysqlFamily && quoted != `"s"."t"` {
			t.Errorf("%s: quoteRef = %q, want double quotes", tc.driver, quoted)
		}
	}
	if _, err := sqlDriver("oracle"); err == nil {
		t.Error("an unknown driver must be refused")
	}
}

// A DSN is built per driver: pgx takes a URL, the mysql driver its own format, and SQLite is a file path.
func TestDSNPerDriver(t *testing.T) {
	pg, err := dsn(entity.Connection{Driver: constants.DriverPostgres, Host: "db", Port: 5432, User: "u", Password: "p", Database: "demo"})
	if err != nil || !strings.HasPrefix(pg, "postgres://u:p@db:5432/demo") {
		t.Errorf("postgres dsn = %q, %v", pg, err)
	}
	for _, driver := range []string{constants.DriverMySQL, constants.DriverMariaDB} {
		got, err := dsn(entity.Connection{Driver: driver, Host: "db", Port: 3306, User: "u", Password: "p", Database: "demo"})
		if err != nil || !strings.Contains(got, "tcp(db:3306)/demo") {
			t.Errorf("%s dsn = %q, %v", driver, got, err)
		}
	}
	if got, err := dsn(entity.Connection{Driver: constants.DriverSQLite, Database: "/tmp/app.db"}); err != nil || got != "/tmp/app.db" {
		t.Errorf("sqlite dsn = %q, %v", got, err)
	}
}
