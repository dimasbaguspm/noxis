package modules

import (
	"errors"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

func TestLoadModuleByType(t *testing.T) {
	sqlFlow := sql.NewChanges(nil, nil, nil, nil)
	mongoFlow := mongodb.NewChanges(nil, nil, nil, nil)
	r := registry{changes: interfaces.ChangeFlows{SQL: sqlFlow, Mongo: mongoFlow}}

	var gotSQL interfaces.SQLChangeFlow
	if err := r.LoadModule(&gotSQL); err != nil {
		t.Fatal(err)
	}
	if gotSQL != sqlFlow {
		t.Fatalf("SQL flow = %T, want registered flow", gotSQL)
	}

	var gotMongo interfaces.MongoChangeFlow
	if err := r.LoadModule(&gotMongo); err != nil {
		t.Fatal(err)
	}
	if gotMongo != mongoFlow {
		t.Fatalf("Mongo flow = %T, want registered flow", gotMongo)
	}

	var gotFlows interfaces.ChangeFlows
	if err := r.LoadModule(&gotFlows); err != nil {
		t.Fatal(err)
	}
	if gotFlows.SQL != sqlFlow || gotFlows.Mongo != mongoFlow {
		t.Fatalf("change flows = %#v, want registered flows", gotFlows)
	}
}

func TestLoadModuleRejectsMissingOrUnsupportedTargets(t *testing.T) {
	r := registry{}
	var sqlFlow interfaces.SQLChangeFlow
	if err := r.LoadModule(&sqlFlow); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("missing SQL flow error = %v, want unsupported", err)
	}
	if err := r.LoadModule(new(string)); !errors.Is(err, constants.ErrUnsupported) {
		t.Fatalf("unknown target error = %v, want unsupported", err)
	}
	if err := r.LoadModule((*interfaces.SQLChangeFlow)(nil)); err == nil {
		t.Fatal("nil target accepted")
	}
}
