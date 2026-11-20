// Package sqlite is the state file: the only durable store in this build, and the only
// place in the tree where SQL is written.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"os"
	"path/filepath"
	"time"

	"modernc.org/sqlite" // registers the "sqlite" driver
)

// SchemaVersion is the schema this build knows how to read. A file with another version is
// refused with a reset instruction, never migrated: a half-migrated journal is worse than a
// refusal, and this file is an audit trail before it is a cache.
//
// v1: targets and journal. v2: targets carry what the last probe measured. v3: proposals and
// their rehearsals. v4: a proposal carries the apply's progress and receipt.
const SchemaVersion = 4

// DB is the state file.
type DB struct {
	db *sql.DB
	// path is kept for error messages: a refusal names the file it refused.
	path string
}

var (
	_ ports.Targets = (*DB)(nil)
	_ ports.Journal = (*DB)(nil)
)

// Open opens the state file, creating it when it is absent, and settles its schema.
func Open(ctx context.Context, path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("state: create %s: %w", dir, err)
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("state: open %s: %w", path, err)
	}
	// One connection: SQLite serialises writes anyway, and a single writer makes the
	// "claim then act" pattern behind apply honest.
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("state: open %s: %w", path, err)
	}
	state := &DB{db: db, path: path}
	if err := state.settle(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return state, nil
}

// Close releases the file.
func (d *DB) Close() error { return d.db.Close() }

// settle creates the schema on an empty file and refuses anything it did not write.
func (d *DB) settle(ctx context.Context) error {
	version, err := d.userVersion(ctx)
	if err != nil {
		return err
	}
	switch {
	case version == SchemaVersion:
		return nil
	case version != 0:
		return fmt.Errorf("state: %s is schema v%d and this build speaks v%d: start with a fresh state file, this build does not migrate one", d.path, version, SchemaVersion)
	}
	// A version-zero file that already holds tables is somebody else's database.
	if err := d.requireEmpty(ctx); err != nil {
		return err
	}
	for _, statement := range schema {
		if _, err := d.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("state: create schema: %w", err)
		}
	}
	if _, err := d.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)); err != nil {
		return fmt.Errorf("state: stamp schema version: %w", err)
	}
	return nil
}

func (d *DB) userVersion(ctx context.Context) (int, error) {
	var version int
	if err := d.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("state: read schema version: %w", err)
	}
	return version, nil
}

func (d *DB) requireEmpty(ctx context.Context) error {
	var tables int
	err := d.db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tables)
	if err != nil {
		return fmt.Errorf("state: inspect %s: %w", d.path, err)
	}
	if tables > 0 {
		return fmt.Errorf("state: %s already holds %d table(s) and no noxis schema version: it is not a state file this build wrote", d.path, tables)
	}
	return nil
}

var schema = []string{
	`CREATE TABLE targets (
		ref             TEXT    PRIMARY KEY,
		name            TEXT    NOT NULL UNIQUE,
		driver          TEXT    NOT NULL,
		host            TEXT    NOT NULL,
		port            INTEGER NOT NULL,
		database        TEXT    NOT NULL,
		username        TEXT    NOT NULL,
		password_cipher TEXT    NOT NULL DEFAULT '',
		tls_mode        TEXT    NOT NULL,
		options         TEXT    NOT NULL DEFAULT '{}',
		created_by      TEXT    NOT NULL,
		archived_at     INTEGER,
		server_version  TEXT    NOT NULL DEFAULT '',
		last_probe_at   INTEGER,
		last_probe_status TEXT  NOT NULL DEFAULT '',
		created_at      INTEGER NOT NULL,
		updated_at      INTEGER NOT NULL
	) STRICT`,
	`CREATE INDEX targets_driver ON targets (driver)`,
	`CREATE TABLE journal (
		ref          TEXT    PRIMARY KEY,
		subject_ref  TEXT    NOT NULL,
		subject_kind TEXT    NOT NULL,
		kind         TEXT    NOT NULL,
		actor        TEXT    NOT NULL,
		detail       TEXT    NOT NULL DEFAULT '{}',
		created_at   INTEGER NOT NULL
	) STRICT`,
	`CREATE INDEX journal_subject ON journal (subject_ref, ref)`,
	`CREATE TABLE proposals (
		ref           TEXT    PRIMARY KEY,
		target_ref    TEXT    NOT NULL REFERENCES targets (ref),
		title         TEXT    NOT NULL,
		kind          TEXT    NOT NULL,
		query         TEXT    NOT NULL,
		rollback      TEXT    NOT NULL DEFAULT '',
		note          TEXT    NOT NULL DEFAULT '',
		status        TEXT    NOT NULL,
		tier          TEXT    NOT NULL,
		irreversible  INTEGER NOT NULL DEFAULT 0,
		proposed_by   TEXT    NOT NULL,
		rehearsal_ref TEXT    NOT NULL DEFAULT '',
		decided_by    TEXT    NOT NULL DEFAULT '',
		decided_at    INTEGER,
		decided_note  TEXT    NOT NULL DEFAULT '',
		applied_by    TEXT    NOT NULL DEFAULT '',
		applied_at    INTEGER,
		failure       TEXT    NOT NULL DEFAULT '',
		started_at    INTEGER,
		progress_stage TEXT  NOT NULL DEFAULT '',
		progress_percent INTEGER NOT NULL DEFAULT -1,
		algorithm     TEXT    NOT NULL DEFAULT '',
		rows_estimated INTEGER NOT NULL DEFAULT 0,
		rows_affected INTEGER NOT NULL DEFAULT 0,
		created_at    INTEGER NOT NULL,
		updated_at    INTEGER NOT NULL
	) STRICT`,
	`CREATE INDEX proposals_target ON proposals (target_ref, ref)`,
	`CREATE INDEX proposals_status ON proposals (status, ref)`,
	`CREATE TABLE rehearsals (
		ref          TEXT    PRIMARY KEY,
		proposal_ref TEXT    NOT NULL REFERENCES proposals (ref),
		tier         TEXT    NOT NULL,
		status       TEXT    NOT NULL,
		schema_diff  TEXT    NOT NULL DEFAULT '',
		row_diff     TEXT    NOT NULL DEFAULT '',
		plan         TEXT    NOT NULL DEFAULT '',
		irreversible INTEGER NOT NULL DEFAULT 0,
		note         TEXT    NOT NULL DEFAULT '',
		duration_ms  INTEGER NOT NULL DEFAULT 0,
		error        TEXT    NOT NULL DEFAULT '',
		created_at   INTEGER NOT NULL
	) STRICT`,
	`CREATE INDEX rehearsals_proposal ON rehearsals (proposal_ref, ref)`,
}

// millis is how every timestamp is stored: an integer, so ordering never depends on a
// string format.
func millis(t time.Time) int64 { return t.UTC().UnixMilli() }

func fromMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

// nullableMillis encodes an absent time as NULL rather than the zero time.
func nullableMillis(t *time.Time) any {
	if t == nil {
		return nil
	}
	return millis(*t)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		// 2067 is SQLITE_CONSTRAINT_UNIQUE.
		return se.Code() == 2067
	}
	return false
}

// notFound turns the driver's empty result into the sentinel the transport maps to 404.
func notFound(err error, what, ref string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s %s", domain.ErrNotFound, what, ref)
	}
	return err
}
