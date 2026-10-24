package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCreatesCurrentVersionedSchema(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var version int
	if err := store.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
	for _, table := range []string{"changes", "connections", "permissions", "users", "access_requests", "access_events"} {
		var found int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found != 1 {
			t.Errorf("table %q missing", table)
		}
	}
	for _, index := range []string{
		"connections_id", "users_email", "users_username", "users_token_hash", "access_requests_pending",
		"changes_by_created", "changes_by_connection_created", "changes_by_schedule", "permissions_by_member",
		"permissions_groups", "access_requests_by_subject", "access_requests_by_status_connection",
		"access_events_by_subject", "access_events_by_actor", "access_events_by_connection",
	} {
		var found int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found != 1 {
			t.Errorf("index %q missing", index)
		}
	}
}

func TestOpenRejectsUnversionedExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE connection_grants (connection TEXT)`); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), path, ""); err == nil || !strings.Contains(err.Error(), "reset NOXIS_DB") {
		t.Fatalf("open legacy database error = %v, want reset instruction", err)
	}
}

func TestOpenRejectsUnknownSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`PRAGMA user_version = 99`); err != nil {
		raw.Close()
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), path, ""); err == nil || !strings.Contains(err.Error(), "schema version 99 is unsupported") {
		t.Fatalf("open unknown version error = %v", err)
	}
}

func TestCanonicalSchemaRejectsInvalidRows(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cases := []struct {
		name  string
		query string
		args  []any
	}{
		{"connection port", `INSERT INTO connections (id, name, driver, port, database, created_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"c1", "c1", "postgres", 65536, "demo", 1}},
		{"connection params JSON", `INSERT INTO connections (id, name, driver, port, database, params, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, []any{"c2", "c2", "postgres", 5432, "demo", "{invalid", 1}},
		{"change kind", `INSERT INTO changes (id, connection, notes, query, rollback, kind, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, []any{"ch1", "demo", "n", "q", "r", "other", "validated", 1}},
		{"change status", `INSERT INTO changes (id, connection, notes, query, rollback, kind, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, []any{"ch2", "demo", "n", "q", "r", "dml", "other", 1}},
		{"permission level", `INSERT INTO permissions (connection, member, member_type, level, expires_at, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, []any{"demo", "user", "user", "owner", 10, "root", 1, 1}},
		{"user role", `INSERT INTO users (subject, role, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, []any{"local:bad", "owner", "active", 1, 1}},
		{"request status", `INSERT INTO access_requests (id, subject, connection, level, status, requested_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"r1", "user", "demo", "read", "other", 1}},
		{"event action", `INSERT INTO access_events (id, at, actor, action) VALUES (?, ?, ?, ?)`, []any{"e1", 1, "root", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := store.db.Exec(tc.query, tc.args...); err == nil {
				t.Fatal("invalid row was accepted")
			}
		})
	}
}
