package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"noxis/internal/entity"
)

// TestPermissionsRoundTrip locks the policy store: upsert into the same row,
// ordering, rejection of an unknown level or member type, and deletion.
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

	if err := store.DeletePermission(ctx, "demo", "dev"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = store.ListPermissions(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("after delete permissions=%+v", list)
	}
}

// TestMigrateGrantsToPermissions opens a pre-rename database (old
// connection_grants table) and proves Open moves the rows into permissions as
// groups and drops the old table.
func TestMigrateGrantsToPermissions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE connection_grants (
			connection TEXT NOT NULL,
			group_name TEXT NOT NULL,
			level      TEXT NOT NULL,
			PRIMARY KEY (connection, group_name)
		);
		INSERT INTO connection_grants VALUES ('demo', 'ops', 'admin');`); err != nil {
		raw.Close()
		t.Fatalf("seed legacy: %v", err)
	}
	raw.Close()

	store, err := Open(ctx, path, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	list, err := store.ListPermissions(ctx, "demo")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Member != "ops" || list[0].MemberType != entity.MemberGroup || list[0].Level != entity.LevelAdmin {
		t.Fatalf("migrated permissions=%+v", list)
	}

	var name string
	err = store.db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='connection_grants'`).Scan(&name)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("connection_grants must be dropped, err=%v", err)
	}
}
