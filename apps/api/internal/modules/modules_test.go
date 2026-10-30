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

func TestLoadModuleByType(t *testing.T) {
	sqlFlow := postgres.NewChanges(nil, nil, nil, nil)
	mongoFlow := mongodb.NewChanges(nil, nil, nil, nil)
	r := registry{mods: map[string]Registration{
		constants.DriverPostgres: {Changes: sqlFlow},
		constants.DriverMongo:    {Changes: mongoFlow},
	}}

	var gotSQL interfaces.SQLChangeFlow
	ctx := common.WithModuleDriver(context.Background(), constants.DriverPostgres)
	if err := r.LoadModule(ctx, &gotSQL); err != nil {
		t.Fatal(err)
	}
	if gotSQL != sqlFlow {
		t.Fatalf("SQL flow = %T, want registered flow", gotSQL)
	}

	var gotMongo interfaces.MongoChangeFlow
	ctx = common.WithModuleDriver(context.Background(), constants.DriverMongo)
	if err := r.LoadModule(ctx, &gotMongo); err != nil {
		t.Fatal(err)
	}
	if gotMongo != mongoFlow {
		t.Fatalf("Mongo flow = %T, want registered flow", gotMongo)
	}
}

func TestLoadEngineModuleUsesContextDriver(t *testing.T) {
	sq, mongo, cache := postgres.New(), mongodb.New(), redis.New()
	r := registry{mods: map[string]Registration{
		constants.DriverPostgres: {Engine: sq},
		constants.DriverMongo:    {Engine: mongo},
		constants.DriverRedis:    {Engine: cache},
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
	var sqlFlow interfaces.SQLChangeFlow
	if err := r.LoadModule(context.Background(), &sqlFlow); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("missing SQL flow error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), new(string)); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("unknown target error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), (*interfaces.SQLChangeFlow)(nil)); err == nil {
		t.Fatal("nil target accepted")
	}
}

func TestDriversAndQueues(t *testing.T) {
	r := New(map[string]Registration{
		constants.DriverPostgres: {Changes: postgres.NewChanges(nil, nil, nil, nil)},
		constants.DriverRedis:    {},
	})
	if got := r.Drivers(); len(got) != 2 || got[0] != constants.DriverPostgres || got[1] != constants.DriverRedis {
		t.Fatalf("drivers = %v, want the registered names sorted", got)
	}
	if _, err := r.Changes(constants.DriverRedis); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("cache queue error = %v, want unsupported", err)
	}
	if _, err := r.Changes(constants.DriverMySQL); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("unregistered queue error = %v, want unsupported", err)
	}
}
