package controller

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

func authzDeps(grants ...entity.Grant) (Deps, interfaces.Module) {
	d, _, _ := newDeps(&fakeSQL{})
	store := &fakeGrants{grants: map[string]entity.Grant{}}
	for _, g := range grants {
		store.grants[g.Group] = g
	}
	d.Grants = store
	return d, d.Modules[constants.FamilySQL]
}

func as(principal entity.Principal) context.Context {
	return WithPrincipal(context.Background(), principal)
}

// TestAuthorizeByEffectiveLevel: the strongest matching grant decides, and a
// read grant never reaches a write or an admin op.
func TestAuthorizeByEffectiveLevel(t *testing.T) {
	d, _ := authzDeps(
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
		ctx := as(entity.Principal{Subject: "alice", Groups: tc.groups})
		err := d.authorize(ctx, "demo", tc.need)
		if tc.ok && err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, errForbidden) {
			t.Errorf("%s: err=%v want forbidden", tc.name, err)
		}
	}
}

// TestAuthorizeOpenModeAndAdmin: no principal (auth off) and a global admin
// both bypass the per-connection check.
func TestAuthorizeOpenModeAndAdmin(t *testing.T) {
	d, _ := authzDeps()
	if err := d.authorize(context.Background(), "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("open mode: %v", err)
	}
	ctx := as(entity.Principal{Subject: "root", Groups: []string{"whatever"}, Admin: true})
	if err := d.authorize(ctx, "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("global admin: %v", err)
	}
}

// TestConnectionsListHidesUnreadable: a caller without a read grant does not
// see the instance.
func TestConnectionsListHidesUnreadable(t *testing.T) {
	d, mod := authzDeps(entity.Grant{Connection: "demo", Group: "readers", Level: entity.LevelRead})

	visible, err := d.connections(as(entity.Principal{Groups: []string{"readers"}}), mod)
	if err != nil || len(visible) != 1 {
		t.Fatalf("granted reader: list=%v err=%v", visible, err)
	}

	hidden, err := d.connections(as(entity.Principal{Groups: []string{"strangers"}}), mod)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("stranger: list=%v err=%v", hidden, err)
	}
}
