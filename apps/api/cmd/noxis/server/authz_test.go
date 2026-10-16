package server

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

func authzBase(grants ...entity.Permission) (testBase, interfaces.Module) {
	base, _, _ := newTestBase(&fakeSQL{})
	store := &fakePermissions{permissions: map[string]entity.Permission{}}
	for _, g := range grants {
		store.permissions[g.Member] = g
	}
	base.Permissions = store
	return base, base.Modules[constants.FamilySQL]
}

func asPrincipal(principal entity.Identity) context.Context {
	return usecases.WithPrincipal(context.Background(), principal)
}

func TestAuthorizeByEffectiveLevel(t *testing.T) {
	base, _ := authzBase(
		entity.Permission{Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead},
		entity.Permission{Connection: "demo", Member: "writers", MemberType: entity.MemberGroup, Level: entity.LevelWrite},
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
		ctx := asPrincipal(entity.Identity{Subject: "alice", Groups: tc.groups})
		err := usecases.Authorize(ctx, base.Permissions, "demo", tc.need)
		if tc.ok && err != nil {
			t.Errorf("%s: err=%v want allowed", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, usecases.ErrForbidden) {
			t.Errorf("%s: err=%v want forbidden", tc.name, err)
		}
	}
}

// TestAuthorizeOpenModeAndAdmin: no principal (auth off) and a global admin
// both bypass the per-connection check.
func TestAuthorizeOpenModeAndAdmin(t *testing.T) {
	base, _ := authzBase()
	if err := usecases.Authorize(context.Background(), base.Permissions, "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("open mode: %v", err)
	}
	ctx := asPrincipal(entity.Identity{Subject: "root", Groups: []string{"whatever"}, Admin: true})
	if err := usecases.Authorize(ctx, base.Permissions, "demo", entity.LevelAdmin); err != nil {
		t.Fatalf("global admin: %v", err)
	}
}

// TestConnectionsListHidesUnreadable: a caller without a read grant does not
// see the instance.
func TestConnectionsListHidesUnreadable(t *testing.T) {
	base, mod := authzBase(entity.Permission{Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead})

	visible, err := (usecases.ConnectionList{Connections: base.Connections, Permissions: base.Permissions}).Run(asPrincipal(entity.Identity{Groups: []string{"readers"}}), mod)
	if err != nil || len(visible) != 1 {
		t.Fatalf("granted reader: list=%v err=%v", visible, err)
	}

	hidden, err := (usecases.ConnectionList{Connections: base.Connections, Permissions: base.Permissions}).Run(asPrincipal(entity.Identity{Groups: []string{"strangers"}}), mod)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("stranger: list=%v err=%v", hidden, err)
	}
}
