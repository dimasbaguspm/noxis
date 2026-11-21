package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A fresh file is created by running every migration, and it says which version it reached.
func TestOpenMigratesAFreshFile(t *testing.T) {
	state := open(t)

	version, err := state.userVersion(context.Background())
	if err != nil {
		t.Fatalf("userVersion: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("version = %d, want %d", version, SchemaVersion)
	}
	for _, table := range []string{"targets", "journal", "proposals", "rehearsals"} {
		if !state.hasTable(t, table) {
			t.Fatalf("table %q is missing after the migration", table)
		}
	}
}

// Opening the same file twice is not a second migration: a restart must not try to create a table that
// is already there.
func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	for range 3 {
		state, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if err := state.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

// A file written by a newer build is refused by name, and the refusal tells the operator what to do.
// Migrating it downward would mean guessing what the newer schema meant.
func TestOpenRefusesANewerSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")

	state, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := state.db.ExecContext(ctx, "PRAGMA user_version = 99"); err != nil {
		t.Fatalf("stamp future version: %v", err)
	}
	if err := state.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := Open(ctx, path); err == nil {
		t.Fatal("a file from a newer build opened, want a refusal")
	} else if !strings.Contains(err.Error(), "upgrade noxis") {
		t.Fatalf("error = %q, want it to say to upgrade", err)
	}
}

// A version-zero file that already holds tables is not a state file: migrating it would create tables
// beside somebody else's.
func TestOpenRefusesAForeignFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")

	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.ExecContext(ctx, "CREATE TABLE someone_elses (id INTEGER)"); err != nil {
		t.Fatalf("create foreign table: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	if _, err := Open(ctx, path); err == nil {
		t.Fatal("a foreign database opened, want a refusal")
	} else if !strings.Contains(err.Error(), "not a state file") {
		t.Fatalf("error = %q, want it to say this is not a state file", err)
	}
}

// A step runs once and stamps its own version: opening a file one step behind brings it forward without
// replaying the steps it already ran.
func TestMigrationsRunInOrderOnceEach(t *testing.T) {
	ctx := context.Background()
	state := open(t) // at SchemaVersion, so the real history has already run

	original := migrations
	t.Cleanup(func() { migrations = original })
	migrations = append(original,
		migration{version: SchemaVersion + 1, stamp: "PRAGMA user_version = 2", statements: []string{`CREATE TABLE v2_marker (id INTEGER) STRICT`}},
		migration{version: SchemaVersion + 2, stamp: "PRAGMA user_version = 3", statements: []string{`CREATE TABLE v3_marker (id INTEGER) STRICT`}},
	)

	if err := state.migrate(ctx, SchemaVersion); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !state.hasTable(t, "v2_marker") || !state.hasTable(t, "v3_marker") {
		t.Fatal("both steps should have run")
	}
	if version, _ := state.userVersion(ctx); version != SchemaVersion+2 {
		t.Fatalf("version = %d, want %d: the last step's stamp", version, SchemaVersion+2)
	}

	// Nothing left to do: a second call must not try to create the tables again.
	if err := state.migrate(ctx, SchemaVersion+2); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// A step that fails leaves the file where it was: no half-migrated journal.
func TestAFailedMigrationRollsBack(t *testing.T) {
	ctx := context.Background()
	state := open(t)

	original := migrations
	t.Cleanup(func() { migrations = original })
	migrations = append(original, migration{
		version: SchemaVersion + 1,
		stamp:   "PRAGMA user_version = 2",
		statements: []string{
			`CREATE TABLE will_be_rolled_back (id INTEGER) STRICT`,
			`CREATE TABLE this is not sql`,
		},
	})

	if err := state.migrate(ctx, SchemaVersion); err == nil {
		t.Fatal("a broken step reported success")
	}
	if state.hasTable(t, "will_be_rolled_back") {
		t.Fatal("the first statement of a failed step survived: the migration is not transactional")
	}
	if version, _ := state.userVersion(ctx); version != SchemaVersion {
		t.Fatalf("version = %d, want %d unchanged", version, SchemaVersion)
	}
}

func (d *DB) hasTable(t *testing.T, name string) bool {
	t.Helper()
	var found string
	err := d.db.QueryRowContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&found)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("hasTable(%q): %v", name, err)
	}
	return found == name
}

// Every step stamps the version it claims: the runner reads that number on the next boot, so a step whose
// stamp disagrees with its version is a file that migrates itself twice. The versions only ever grow.
func TestEveryMigrationStampsItsVersion(t *testing.T) {
	previous := 0
	for _, step := range migrations {
		if step.version <= previous {
			t.Fatalf("version %d follows %d: the history must grow", step.version, previous)
		}
		if !strings.HasSuffix(step.stamp, "= "+strconv.Itoa(step.version)) {
			t.Fatalf("step %d stamps %q, which is not its version", step.version, step.stamp)
		}
		if len(step.statements) == 0 {
			t.Fatalf("step %d has no statements", step.version)
		}
		previous = step.version
	}
	if previous != SchemaVersion {
		t.Fatalf("SchemaVersion = %d, the last step reaches %d", SchemaVersion, previous)
	}
}
