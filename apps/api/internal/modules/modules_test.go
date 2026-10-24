package modules

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/common"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

func TestLoadModuleByType(t *testing.T) {
	sqlFlow := sql.NewChanges(nil, nil, nil, nil)
	mongoFlow := mongodb.NewChanges(nil, nil, nil, nil)
	r := registry{changes: interfaces.ChangeFlows{SQL: sqlFlow, Mongo: mongoFlow}}

	var gotSQL interfaces.SQLChangeFlow
	if err := r.LoadModule(context.Background(), &gotSQL); err != nil {
		t.Fatal(err)
	}
	if gotSQL != sqlFlow {
		t.Fatalf("SQL flow = %T, want registered flow", gotSQL)
	}

	var gotMongo interfaces.MongoChangeFlow
	if err := r.LoadModule(context.Background(), &gotMongo); err != nil {
		t.Fatal(err)
	}
	if gotMongo != mongoFlow {
		t.Fatalf("Mongo flow = %T, want registered flow", gotMongo)
	}

	var gotFlows interfaces.ChangeFlows
	if err := r.LoadModule(context.Background(), &gotFlows); err != nil {
		t.Fatal(err)
	}
	if gotFlows.SQL != sqlFlow || gotFlows.Mongo != mongoFlow {
		t.Fatalf("change flows = %#v, want registered flows", gotFlows)
	}
}

func TestLoadEngineModuleUsesContextFamily(t *testing.T) {
	sq, mongo, cache := sql.New(), mongodb.New(), memcache.New()
	r := registry{mods: map[string]interfaces.Module{
		constants.FamilySQL:      sq,
		constants.FamilyMongo:    mongo,
		constants.FamilyMemcache: cache,
	}}

	ctx := common.WithModuleFamily(context.Background(), constants.FamilySQL)
	var gotSQL interfaces.SQLModule
	if err := r.LoadModule(ctx, &gotSQL); err != nil || gotSQL != sq {
		t.Fatalf("SQL module = %T, error = %v", gotSQL, err)
	}

	ctx = common.WithModuleFamily(context.Background(), constants.FamilyMongo)
	var gotMongo interfaces.MongoModule
	if err := r.LoadModule(ctx, &gotMongo); err != nil || gotMongo != mongo {
		t.Fatalf("Mongo module = %T, error = %v", gotMongo, err)
	}

	ctx = common.WithModuleFamily(context.Background(), constants.FamilyMemcache)
	var gotCache interfaces.MemcacheModule
	if err := r.LoadModule(ctx, &gotCache); err != nil || gotCache != cache {
		t.Fatalf("cache module = %T, error = %v", gotCache, err)
	}

	var wrong interfaces.SQLModule
	if err := r.LoadModule(ctx, &wrong); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("wrong-family error = %v, want unsupported", err)
	}
	if err := r.LoadModule(context.Background(), &wrong); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("missing-family error = %v, want unsupported", err)
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
