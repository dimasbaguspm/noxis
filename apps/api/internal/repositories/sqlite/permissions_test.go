package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// TestPermissionsRoundTrip locks the policy store: upsert into the same row, ordering, rejection of an unknown level or member type, and deletion.
func TestPermissionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for _, p := range []entity.Permission{
		{Connection: "demo", Member: "dev", MemberType: entity.MemberGroup, Level: entity.LevelRead},
		{Connection: "demo", Member: "dev", MemberType: entity.MemberGroup, Level: entity.LevelWrite}, // upsert
		{Connection: "demo", Member: "ops", MemberType: entity.MemberGroup, Level: entity.LevelAdmin},
		{Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser, Level: entity.LevelRead},
	} {
		if err := store.SetPermission(ctx, p); err != nil {
			t.Fatalf("set %+v: %v", p, err)
		}
	}

	list, err := store.ListPermissions(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// ordered by member: dev, local:alice, ops
	if len(list) != 3 || list[0].Member != "dev" || list[0].Level != entity.LevelWrite {
		t.Fatalf("permissions=%+v", list)
	}
	if list[1].Member != "local:alice" || list[1].MemberType != entity.MemberUser {
		t.Fatalf("permissions=%+v", list)
	}

	if err := store.SetPermission(ctx, entity.Permission{Connection: "demo", Member: "x", MemberType: entity.MemberGroup, Level: "root"}); err == nil {
		t.Fatal("unknown level must be rejected")
	}
	if err := store.SetPermission(ctx, entity.Permission{Connection: "demo", Member: "x", MemberType: "team", Level: entity.LevelRead}); err == nil {
		t.Fatal("unknown member type must be rejected")
	}

	if err := store.DeletePermission(ctx, "demo", "dev", "root"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = store.ListPermissions(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("after delete permissions=%+v", list)
	}
	if list[0].ExpiresAt == 0 {
		t.Fatal("a stored grant must carry an expiry")
	}

	events, err := store.ListAccessEvents(ctx, "local:alice", 20)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if !hasAction(events, constants.AccessGranted) {
		t.Fatalf("expected a granted event, got %+v", events)
	}
	revoked, err := store.ListAccessEvents(ctx, "dev", 20)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if !hasAction(revoked, constants.AccessRevoked) {
		t.Fatalf("expected a revoked event, got %+v", revoked)
	}
}

func hasAction(events []entity.AccessEvent, action string) bool {
	for i := range events {
		if events[i].Action == action {
			return true
		}
	}
	return false
}
