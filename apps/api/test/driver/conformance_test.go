// Package driver proves every driver module keeps one contract without sharing code: the five SQL engines, the two cache engines and the document engine.
package driver

import (
	"testing"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/cockroach"
	"noxis/internal/modules/mariadb"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/mysql"
	"noxis/internal/modules/postgres"
	"noxis/internal/modules/redis"
	"noxis/internal/modules/sqlite"
	"noxis/internal/modules/valkey"
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

// TestCacheModulesKeepTheContract walks the two cache engines: each names its own driver, refuses the other, and browses keys instead of claiming it can prove a change.
func TestCacheModulesKeepTheContract(t *testing.T) {
	contracts := []struct {
		driver string
		module interfaces.Module
	}{
		{constants.DriverRedis, redis.New()},
		{constants.DriverValkey, valkey.New()},
	}
	for _, c := range contracts {
		if got := c.module.Driver(); got != c.driver {
			t.Errorf("%s: Driver() = %q", c.driver, got)
		}
		for _, other := range []string{constants.DriverRedis, constants.DriverValkey, constants.DriverPostgres, "oracle"} {
			if got, want := c.module.Supports(other), other == c.driver; got != want {
				t.Errorf("%s: Supports(%q) = %v, want %v", c.driver, other, got, want)
			}
		}
		if _, ok := c.module.(interfaces.KeyBrowser); !ok {
			t.Errorf("%s: %T does not browse keys", c.driver, c.module)
		}
		if _, ok := c.module.(interfaces.SQLChangeEngine); ok {
			t.Errorf("%s: %T claims to prove SQL", c.driver, c.module)
		}
	}
}

// TestDocumentModuleKeepsTheContract proves the document engine names its driver, browses documents and drives its own change queue.
func TestDocumentModuleKeepsTheContract(t *testing.T) {
	var module interfaces.Module = mongodb.New()
	if got := module.Driver(); got != constants.DriverMongo {
		t.Errorf("Driver() = %q", got)
	}
	if !module.Supports(constants.DriverMongo) || module.Supports(constants.DriverRedis) {
		t.Error("Supports does not follow the driver")
	}
	if _, ok := module.(interfaces.MongoChangeEngine); !ok {
		t.Errorf("%T does not browse documents", module)
	}
	var queue interfaces.ChangeFlow = mongodb.NewChanges(nil, nil, nil, nil)
	if got := queue.Driver(); got != constants.DriverMongo {
		t.Errorf("its change queue reports %q", got)
	}
	if _, ok := queue.(interfaces.MongoChangeFlow); !ok {
		t.Errorf("%T cannot submit a document request", queue)
	}
}
