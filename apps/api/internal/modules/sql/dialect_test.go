package sql

import (
	"testing"

	"noxis/internal/constants"
)

// TestEveryDriverHasACompleteProfile fails when a driver constant has no entry, or an entry leaves out a field a feature reads. That is what stops a driver being half-added: the compiler cannot see the gap, this test can.
func TestEveryDriverHasACompleteProfile(t *testing.T) {
	drivers := []string{
		constants.DriverPostgres,
		constants.DriverMySQL,
		constants.DriverMariaDB,
		constants.DriverSQLite,
		constants.DriverCockroach,
	}
	for _, name := range drivers {
		d, ok := dialectByName[name]
		if !ok {
			t.Fatalf("driver %q has no dialect entry", name)
		}
		if d.name == "" || d.driver == "" || d.quote == 0 || d.analyze == "" || d.dsn == nil {
			t.Errorf("%s: profile is missing a required field", name)
		}
		if d.maxOpenConns <= 0 {
			t.Errorf("%s: maxOpenConns must be positive", name)
		}
		if d.catalogPerTable {
			// a per-table catalog reads pragmas, so it carries no information_schema query
			continue
		}
		// estimates and comments are capabilities, not requirements: an engine that reports neither is still browsable, and capabilities() says so out loud.
		if d.keysQuery == "" || d.indexesQuery == "" {
			t.Errorf("%s: the catalog queries the browser depends on are missing", name)
		}
		if d.scanIndex == nil || d.scanEstimate == nil {
			t.Errorf("%s: catalog row decoders missing", name)
		}
	}
	if _, err := dialectOf("oracle"); err == nil {
		t.Error("an unknown driver must be refused")
	}
	if got, err := sqlDriver("postgresql"); err != nil || got != dbDriverPostgres {
		t.Errorf("the long form of postgres must resolve: %q, %v", got, err)
	}
}

// TestTransactionalRule proves the verification rule each engine gets: sqlite never needs a scratch schema, postgres falls back to the live transaction only when a qualified name could escape one, and mysql can only roll back DML.
func TestTransactionalRule(t *testing.T) {
	cases := []struct {
		driver string
		text   string
		want   bool
	}{
		{constants.DriverSQLite, "CREATE TABLE t (id INT)", true},
		{constants.DriverCockroach, "CREATE TABLE t (id INT)", true},
		{constants.DriverPostgres, "UPDATE t SET id = 1", false},
		{constants.DriverPostgres, "UPDATE public.t SET id = 1", true},
		{constants.DriverMySQL, "UPDATE t SET id = 1", true},
		{constants.DriverMySQL, "CREATE TABLE t (id INT)", false},
		{constants.DriverMariaDB, "CREATE TABLE t (id INT)", false},
	}
	for _, tc := range cases {
		d, err := dialectOf(tc.driver)
		if err != nil {
			t.Fatalf("%s: %v", tc.driver, err)
		}
		if got := d.transactional(tc.text); got != tc.want {
			t.Errorf("%s %q: transactional = %v, want %v", tc.driver, tc.text, got, tc.want)
		}
	}
}
