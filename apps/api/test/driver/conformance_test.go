// Package driver proves the five SQL driver modules keep one contract without sharing code.
package driver

import (
	"testing"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/cockroach"
	"noxis/internal/modules/mariadb"
	"noxis/internal/modules/mysql"
	"noxis/internal/modules/postgres"
	"noxis/internal/modules/sqlite"
)

// contract is one module plus the queue it must drive, so every SQL driver is checked the same way.
type contract struct {
	driver string
	module interfaces.Module
	queue  interfaces.ChangeFlow
}

// TestSQLModulesKeepTheContract walks the five modules: each names its own driver, refuses the other four, proves and applies SQL, and drives a change queue that reports the same driver.
func TestSQLModulesKeepTheContract(t *testing.T) {
	contracts := []contract{
		{constants.DriverPostgres, postgres.New(), postgres.NewChanges(nil, nil, nil, nil)},
		{constants.DriverMySQL, mysql.New(), mysql.NewChanges(nil, nil, nil, nil)},
		{constants.DriverMariaDB, mariadb.New(), mariadb.NewChanges(nil, nil, nil, nil)},
		{constants.DriverSQLite, sqlite.New(), sqlite.NewChanges(nil, nil, nil, nil)},
		{constants.DriverCockroach, cockroach.New(), cockroach.NewChanges(nil, nil, nil, nil)},
	}
	every := []string{
		constants.DriverPostgres,
		constants.DriverCockroach,
		constants.DriverMySQL,
		constants.DriverMariaDB,
		constants.DriverSQLite,
	}

	seen := map[string]bool{}
	for _, c := range contracts {
		if c.driver == "" {
			t.Fatal("a module registers under an empty driver")
		}
		if seen[c.driver] {
			t.Fatalf("driver %q is claimed twice", c.driver)
		}
		seen[c.driver] = true

		if got := c.module.Driver(); got != c.driver {
			t.Errorf("%s: Driver() = %q", c.driver, got)
		}
		for _, other := range every {
			if got, want := c.module.Supports(other), other == c.driver; got != want {
				t.Errorf("%s: Supports(%q) = %v, want %v", c.driver, other, got, want)
			}
		}
		if c.module.Supports("oracle") {
			t.Errorf("%s: Supports(an unknown driver) = true", c.driver)
		}
		if _, ok := c.module.(interfaces.SQLChangeEngine); !ok {
			t.Errorf("%s: %T does not prove and apply SQL", c.driver, c.module)
		}
		if got := c.queue.Driver(); got != c.driver {
			t.Errorf("%s: its change queue reports %q", c.driver, got)
		}
		if _, ok := c.queue.(interfaces.SQLChangeFlow); !ok {
			t.Errorf("%s: %T cannot submit a SQL change", c.driver, c.queue)
		}
	}
	if len(seen) != len(every) {
		t.Fatalf("registered %d drivers, want %d", len(seen), len(every))
	}
}
