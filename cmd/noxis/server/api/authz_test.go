package api

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/application"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

func authzApp(grants ...entity.Grant) (*application.App, interfaces.Module) {
	app, _, _ := newApp(&fakeSQL{})
	store := &fakeGrants{grants: map[string]entity.Grant{}}
	for _, g := range grants {
		store.grants[g.Group] = g
	}
	app.Grants = store
	return app, app.Modules[constants.FamilySQL]
}

func asPrincipal(principal entity.Principal) context.Context {
	return application.WithPrincipal(context.Background(), principal)
}

func TestAuthorizeByEffectiveLevel(t *testing.T) {
	app, _ := authzApp(
		entity.Grant{Connection: "demo", Group: "readers", Level: entity.LevelRead},
		entity.Grant{Connection: "demo", Group: "writers", Level: entity.LevelWrite},
	)

	cases := []struct {
		name   string
		groups []string
		need   string
		ok     bool
	}{
		{"reader reads", []string{"readers"}, entity.LevelRead, true},
		{"reader cannot write", []string{"readers"}, entity.LevelWrite, false},
		{"writer writes", []string{"writers"}, entity.LevelWrite, true},
		{"writer cannot administer", []string{"writers"}, entity.LevelAdmin, false},
		{"unknown group reads nothing", []string{"nobody"}, entity.LevelRead, false},
		{"strongest wins", []string{"readers", "writers"}, entity.LevelWrite, true},
	}
	for _, tc := range cases {
		ctx := asPrincipal(entity.Principal{Subject: "alice", Groups: tc.groups})
		err := app.Authorize(ctx, "demo", tc.need)
		if tc.ok && err != nil {
			t.Errorf("%s: err=%v want allowed", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, application.ErrForbidden) {
			t.Errorf("%s: err=%v want forbidden", tc.name, err)
		}
	}
}

// TestAuthorizeOpenModeAndAdmin: no principal (auth off) and a global admin
// both bypass the per-connection check.
func TestAuthorizeOpenModeAndAdmin(t *testing.T) {
	app, _ := authzApp()
	if err := app.Authorize(context.Background(), "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("open mode: %v", err)
	}
	ctx := asPrincipal(entity.Principal{Subject: "root", Groups: []string{"whatever"}, Admin: true})
	if err := app.Authorize(ctx, "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("global admin: %v", err)
	}
}

// TestConnectionsListHidesUnreadable: a caller without a read grant does not
// see the instance.
func TestConnectionsListHidesUnreadable(t *testing.T) {
	app, mod := authzApp(entity.Grant{Connection: "demo", Group: "readers", Level: entity.LevelRead})

	visible, err := app.ConnectionList(asPrincipal(entity.Principal{Groups: []string{"readers"}}), mod)
	if err != nil || len(visible) != 1 {
		t.Fatalf("granted reader: list=%v err=%v", visible, err)
	}

	hidden, err := app.ConnectionList(asPrincipal(entity.Principal{Groups: []string{"strangers"}}), mod)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("stranger: list=%v err=%v", hidden, err)
	}
}