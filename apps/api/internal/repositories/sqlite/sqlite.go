// Package sqlite persists connections and changes (SQLite via modernc,
// cgo-free) — the app's only state.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/secret"
	"noxis/internal/constants"
	"noxis/internal/interfaces"

	// modernc SQLite driver: pure-Go, cgo-free (interface is database/sql).
	_ "modernc.org/sqlite"
)

type Store struct {
	db     *sql.DB
	secret *secret.Cipher
}

// Satisfiers.
var (
	_ interfaces.ConnectionRepository  = (*Store)(nil)
	_ interfaces.ChangeRepository      = (*Store)(nil)
	_ interfaces.PermissionRepository  = (*Store)(nil)
	_ interfaces.UserRepository        = (*Store)(nil)
	_ interfaces.AccessEventRepository = (*Store)(nil)
)

// Open opens the store and verifies it answers a ping. The parent directory
// is created when missing and sqlite creates the file itself, so the app is
// deployable with zero host-side files. An empty secretKey keeps passwords in
// plaintext (open mode); a set key encrypts them at rest.
func Open(ctx context.Context, path, secretKey string) (*Store, error) {
	cipher, err := secret.New(secretKey)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	// The bind-mount trap: docker creates a DIRECTORY at a missing bind
	// target, and sqlite cannot open a directory ("unable to open database
	// file (14)"). Name it instead of crash-looping.
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		return nil, fmt.Errorf("open %s: is a directory — bind-mount a folder (named volume), not a file", path)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("open %s: mkdir: %w", path, err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, secret: cipher}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	// pre-existing volumes: add columns introduced after the first release.
	// Each step is independent and best-effort; duplicates are ignored.
	for _, ddl := range migrations {
		_, _ = db.Exec(ddl)
	}
	// instance ids were added after the first release: backfill existing rows,
	// then enforce uniqueness (an index over the not-yet-filled column would
	// collide on the empty default).
	if err := backfillConnectionIDs(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("backfill connection ids: %w", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS connections_id ON connections(id)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("connection id index: %w", err)
	}
	if err := s.encryptLegacyPasswords(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("encrypt stored passwords: %w", err)
	}
	if err := migrateGrantsToPermissions(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate grants: %w", err)
	}
	if err := backfillAccessExpiry(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("backfill access expiry: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return s, nil
}

// migrateGrantsToPermissions moves a pre-rename connection_grants table into
// permissions once, then drops it. Fresh databases have no such table.
func migrateGrantsToPermissions(ctx context.Context, db *sql.DB) error {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='connection_grants'`).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO permissions (connection, member, member_type, level)
		SELECT connection, group_name, 'group', level FROM connection_grants`); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `DROP TABLE connection_grants`)
	return err
}

// backfillAccessExpiry gives every permission written before the 120-day rule
// an expiry and attributes it to "system". Rows with an expiry are left alone,
// so it is effectively one-time.
func backfillAccessExpiry(ctx context.Context, db *sql.DB) error {
	now := time.Now()
	if _, err := db.ExecContext(ctx,
		`UPDATE permissions SET expires_at = ?, created_by = 'system', created_at = ?, updated_at = ?
		 WHERE expires_at = 0`,
		now.Add(constants.AccessGrantTTL).Unix(), now.Unix(), now.Unix()); err != nil {
		return fmt.Errorf("backfill permission expiry: %w", err)
	}
	return nil
}

// migrations are applied best-effort; duplicates (fresh databases) error and
// are ignored.
var migrations = []string{
	`ALTER TABLE changes ADD COLUMN kind TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE changes ADD COLUMN rejected_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN est_duration_ms INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN est_affected INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN applied_affected INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN applied_duration_ms INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN affected_tables TEXT NOT NULL DEFAULT '[]'`,
	`ALTER TABLE changes ADD COLUMN affected_columns TEXT NOT NULL DEFAULT '[]'`,
	`ALTER TABLE changes ADD COLUMN scheduled_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN created_by TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE connections ADD COLUMN id TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE users ADD COLUMN last_login_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE users ADD COLUMN is_super INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''`,
	// legacy local accounts were namespaced by username: keep it as the email
	// so the account keeps signing in by the name it was created with
	`UPDATE users SET email = username WHERE email = '' AND username != ''`,
	`CREATE UNIQUE INDEX IF NOT EXISTS users_email ON users(email) WHERE email != ''`,
	`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'member'`,
	`UPDATE users SET role = 'super' WHERE is_super = 1`,
	`ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`,
	`UPDATE users SET status = 'banned' WHERE disabled = 1`,
	`ALTER TABLE permissions ADD COLUMN expires_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE permissions ADD COLUMN created_by TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE permissions ADD COLUMN created_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE permissions ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0`,
	`CREATE TABLE IF NOT EXISTS access_requests (
	  id            TEXT PRIMARY KEY,
	  subject       TEXT NOT NULL,
	  connection    TEXT NOT NULL,
	  level         TEXT NOT NULL,
	  reason        TEXT NOT NULL DEFAULT '',
	  status        TEXT NOT NULL,
	  requested_at  INTEGER NOT NULL,
	  decided_by    TEXT NOT NULL DEFAULT '',
	  decided_at    INTEGER NOT NULL DEFAULT 0,
	  decision_note TEXT NOT NULL DEFAULT '',
	  expires_at    INTEGER NOT NULL DEFAULT 0
	)`,
	`CREATE INDEX IF NOT EXISTS access_requests_pending ON access_requests(status, connection)`,
	`CREATE TABLE IF NOT EXISTS access_events (
	  id         TEXT PRIMARY KEY,
	  at         INTEGER NOT NULL,
	  actor      TEXT NOT NULL DEFAULT '',
	  action     TEXT NOT NULL,
	  subject    TEXT NOT NULL DEFAULT '',
	  connection TEXT NOT NULL DEFAULT '',
	  level      TEXT NOT NULL DEFAULT '',
	  request_id TEXT NOT NULL DEFAULT '',
	  note       TEXT NOT NULL DEFAULT ''
	)`,
	`ALTER TABLE changes ADD COLUMN applying_at INTEGER NOT NULL DEFAULT 0`,
}

// backfillConnectionIDs gives every pre-migration row a stable id. New rows
// get one in Save; this only touches legacy volumes.
func backfillConnectionIDs(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT name FROM connections WHERE id = '' OR id IS NULL`)
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		names = append(names, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, name := range names {
		if _, err := db.ExecContext(ctx, `UPDATE connections SET id = ? WHERE name = ?`, idgen.UUID(), name); err != nil {
			return err
		}
	}
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS changes (
  id          TEXT PRIMARY KEY,
  connection  TEXT NOT NULL,
  notes       TEXT NOT NULL,
  query       TEXT NOT NULL,
  rollback    TEXT NOT NULL,
  kind        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  checks      TEXT NOT NULL DEFAULT '[]',
  created_at  INTEGER NOT NULL,
  created_by  TEXT NOT NULL DEFAULT '',
  approved_at INTEGER NOT NULL DEFAULT 0,
  applying_at INTEGER NOT NULL DEFAULT 0,
  applied_at  INTEGER NOT NULL DEFAULT 0,
  rejected_at INTEGER NOT NULL DEFAULT 0,
  est_duration_ms INTEGER NOT NULL DEFAULT 0,
  est_affected    INTEGER NOT NULL DEFAULT 0,
  applied_affected   INTEGER NOT NULL DEFAULT 0,
  applied_duration_ms INTEGER NOT NULL DEFAULT 0,
  scheduled_at INTEGER NOT NULL DEFAULT 0,
  affected_tables  TEXT NOT NULL DEFAULT '[]',
  affected_columns TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS connections (
  name       TEXT PRIMARY KEY,
  id         TEXT NOT NULL DEFAULT '',
  driver     TEXT NOT NULL,
  host       TEXT NOT NULL,
  port       INTEGER NOT NULL,
  user       TEXT NOT NULL,
  password   TEXT NOT NULL DEFAULT '',
  database   TEXT NOT NULL,
  params     TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  created_by TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS permissions (
  connection  TEXT NOT NULL,
  member      TEXT NOT NULL,
  member_type TEXT NOT NULL,
  level       TEXT NOT NULL,
  PRIMARY KEY (connection, member)
);

CREATE TABLE IF NOT EXISTS users (
  subject          TEXT PRIMARY KEY,
  email            TEXT NOT NULL DEFAULT '',
  username         TEXT NOT NULL DEFAULT '',
  password         TEXT NOT NULL DEFAULT '',
  groups           TEXT NOT NULL DEFAULT '[]',
  role             TEXT NOT NULL DEFAULT 'member',
  status           TEXT NOT NULL DEFAULT 'active',
  token_hash       TEXT NOT NULL DEFAULT '',
  token_expires_at INTEGER NOT NULL DEFAULT 0,
  last_login_at    INTEGER NOT NULL DEFAULT 0,
  created_at       INTEGER NOT NULL,
  updated_at       INTEGER NOT NULL
);

-- a local account may be named; OIDC reference rows have no username
CREATE UNIQUE INDEX IF NOT EXISTS users_username ON users(username) WHERE username != '';
-- one active token per account; empty means logged out
CREATE UNIQUE INDEX IF NOT EXISTS users_token_hash ON users(token_hash) WHERE token_hash != '';`

// encryptLegacyPasswords seals rows written before a key was set, so a store
// that gains a key converges to ciphertext. No-op without a key.
func (s *Store) encryptLegacyPasswords(ctx context.Context) error {
	if s.secret == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT name, password FROM connections WHERE password != ''`)
	if err != nil {
		return err
	}
	type legacy struct{ name, password string }
	var pending []legacy
	for rows.Next() {
		var l legacy
		if err := rows.Scan(&l.name, &l.password); err != nil {
			rows.Close()
			return err
		}
		if !secret.Encrypted(l.password) {
			pending = append(pending, l)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, l := range pending {
		sealed, err := s.secret.Encrypt(l.password)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE connections SET password = ? WHERE name = ?`, sealed, l.name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() { _ = s.db.Close() }

// scanner is the shared row/rows interface.
type scanner interface{ Scan(dest ...any) error }

// queryRows runs a query and scans every row with scan (shared by the list
// methods, so the iteration lives in one place).
func queryRows[T any](ctx context.Context, db *sql.DB, q string, scan func(scanner) (T, error)) ([]T, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// WithTx runs fn in a transaction: commit on success, rollback otherwise.
func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
