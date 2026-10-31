package modules

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/common"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/postgres"
	"noxis/internal/modules/redis"
)

// TestLoadModuleFillsTheRequestedCapability proves a handler names the seam it needs and the registry hands back the
// module registered for the driver in the context.
func TestLoadModuleFillsTheRequestedCapability(t *testing.T) {
	sq, mongo, cache := postgres.New(), mongodb.New(), redis.New()
	r := registry{mods: map[string]interfaces.Module{
		constants.DriverPostgres: sq,
		constants.DriverMongo:    mongo,
		constants.DriverRedis:    cache,
	}}

	ctx := common.WithModuleDriver(context.Background(), constants.DriverPostgres)
	var gotSQL interfaces.SQLChangeEngine
	if err := r.LoadModule(ctx, &gotSQL); err != nil || gotSQL != sq {
		t.Fatalf("SQL module = %T, error = %v", gotSQL, err)
	}

	ctx = common.WithModuleDriver(context.Background(), constants.DriverMongo)
	var gotMongo interfaces.MongoChangeEngine
	if err := r.LoadModule(ctx, &gotMongo); err != nil || gotMongo != mongo {
		t.Fatalf("Mongo module = %T, error = %v", gotMongo, err)
	}

	ctx = common.WithModuleDriver(context.Background(), constants.DriverRedis)
	var gotCache interfaces.KeyBrowser
	if err := r.LoadModule(ctx, &gotCache); err != nil || gotCache != cache {
		t.Fatalf("cache module = %T, error = %v", gotCache, err)
	}

	var wrong interfaces.SQLChangeEngine
	if err := r.LoadModule(ctx, &wrong); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("wrong-driver error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), &wrong); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("missing-driver error = %v, want unsupported", err)
	}
}

func TestLoadModuleRejectsMissingOrUnsupportedTargets(t *testing.T) {
	r := registry{}
	var sql interfaces.SQLChangeEngine
	if err := r.LoadModule(context.Background(), &sql); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("missing driver error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), new(string)); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("unknown target error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), (*interfaces.SQLChangeEngine)(nil)); err == nil {
		t.Fatal("nil target accepted")
	}
}

func TestDriversAndEngine(t *testing.T) {
	r := New(map[string]interfaces.Module{
		constants.DriverPostgres: postgres.New(),
		constants.DriverRedis:    redis.New(),
	})
	if got := r.Drivers(); len(got) != 2 || got[0] != constants.DriverPostgres || got[1] != constants.DriverRedis {
		t.Fatalf("drivers = %v, want the registered names sorted", got)
	}
	if _, err := r.Engine(constants.DriverMySQL); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("unregistered engine error = %v, want unsupported", err)
	}
	if module, err := r.Engine(constants.DriverRedis); err != nil || module.Driver() != constants.DriverRedis {
		t.Fatalf("redis engine = %v, %v", module, err)
	}
}

// TestDefaultRegistersEachDriverOnce locks the built-in set: every driver is registered under its own name, so a
// connection is never served by another driver's module.
func TestDefaultRegistersEachDriverOnce(t *testing.T) {
	r := Default()
	for _, driver := range r.Drivers() {
		module, err := r.Engine(driver)
		if err != nil {
			t.Fatalf("%s: %v", driver, err)
		}
		if got := module.Driver(); got != driver {
			t.Errorf("%s is served by the %s module", driver, got)
		}
		if !module.Supports(driver) {
			t.Errorf("%s: Supports(%q) = false", driver, driver)
		}
	}
}
