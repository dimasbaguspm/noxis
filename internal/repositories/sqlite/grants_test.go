package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"noxis/internal/entity"
)

// TestGrantsRoundTrip locks the policy store: upsert into the same row,
// ordering, rejection of an unknown level, and deletion.
func TestGrantsRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "noxis.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for _, g := range []entity.Grant{
		{Connection: "demo", Group: "dev", Level: entity.LevelRead},
		{Connection: "demo", Group: "dev", Level: entity.LevelWrite}, // upsert
		{Connection: "demo", Group: "ops", Level: entity.LevelAdmin},
	} {
		if err := store.SetGrant(ctx, g); err != nil {
			t.Fatalf("set %+v: %v", g, err)
		}
	}

	list, err := store.ListGrants(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Group != "dev" || list[0].Level != entity.LevelWrite || list[1].Group != "ops" {
		t.Fatalf("grants=%+v", list)
	}

	if err := store.SetGrant(ctx, entity.Grant{Connection: "demo", Group: "x", Level: "root"}); err == nil {
		t.Fatal("unknown level must be rejected")
	}

	if err := store.DeleteGrant(ctx, "demo", "dev"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = store.ListGrants(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Group != "ops" {
		t.Fatalf("after delete grants=%+v", list)
	}
}
