// Package sqlite persists connections and changes (SQLite via modernc, cgo-free) — the app's only state.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"noxis/internal/common/secret"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories"

	// modernc SQLite driver: pure-Go, cgo-free (interface is database/sql).
	_ "modernc.org/sqlite"
)

var _ repositories.Store = (*Store)(nil)

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

// Open creates a fresh current-schema database or rejects an incompatible existing file, then verifies the store with a ping.
func Open(ctx context.Context, path, secretKey string) (*Store, error) {
	cipher, err := secret.New(secretKey)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	// Reject missing file bind targets that Docker created as directories; SQLite cannot open them.
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
	if err := initializeSchema(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.encryptPlaintextPasswords(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("encrypt stored passwords: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return s, nil
}

const schemaVersion = 1

func initializeSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read SQLite schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version != 0 {
		return fmt.Errorf("SQLite schema version %d is unsupported; reset NOXIS_DB for version %d", version, schemaVersion)
	}
	var tables int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables); err != nil {
		return fmt.Errorf("inspect SQLite schema: %w", err)
	}
	if tables != 0 {
		return errors.New("existing SQLite schema is unsupported; reset NOXIS_DB to create the current schema")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin SQLite schema: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, schemaSQL()); err != nil {
		return fmt.Errorf("create SQLite schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("set SQLite schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SQLite schema: %w", err)
	}
	return nil
}

const schemaTemplate = `
CREATE TABLE changes (
  id          TEXT PRIMARY KEY CHECK (length(id) > 0),
  connection  TEXT NOT NULL CHECK (length(connection) > 0),
  notes       TEXT NOT NULL CHECK (length(notes) > 0),
  query       TEXT NOT NULL DEFAULT '',
  rollback    TEXT NOT NULL DEFAULT '',
  kind        TEXT NOT NULL CHECK (kind IN (%s)),
  status      TEXT NOT NULL CHECK (status IN (%s)),
  checks      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(checks)),
  created_at  INTEGER NOT NULL CHECK (created_at > 0),
  created_by  TEXT NOT NULL DEFAULT '',
  approved_at INTEGER NOT NULL DEFAULT 0 CHECK (approved_at >= 0),
  approved_by TEXT NOT NULL DEFAULT '',
  applying_at INTEGER NOT NULL DEFAULT 0 CHECK (applying_at >= 0),
  applied_at  INTEGER NOT NULL DEFAULT 0 CHECK (applied_at >= 0),
  applied_by  TEXT NOT NULL DEFAULT '',
  rejected_at INTEGER NOT NULL DEFAULT 0 CHECK (rejected_at >= 0),
  rejected_by TEXT NOT NULL DEFAULT '',
  est_duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (est_duration_ms >= 0),
  est_affected INTEGER NOT NULL DEFAULT 0 CHECK (est_affected >= 0),
  applied_affected INTEGER NOT NULL DEFAULT 0 CHECK (applied_affected >= 0),
  applied_duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (applied_duration_ms >= 0),
  scheduled_at INTEGER NOT NULL DEFAULT 0 CHECK (scheduled_at >= 0),
  scheduled_by TEXT NOT NULL DEFAULT '',
  affected_tables TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(affected_tables)),
  affected_columns TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(affected_columns))
) STRICT;

CREATE TABLE connections (
  name       TEXT PRIMARY KEY CHECK (length(trim(name)) > 0),
  id         TEXT NOT NULL CHECK (length(id) > 0),
  driver     TEXT NOT NULL CHECK (length(driver) > 0),
  host       TEXT NOT NULL DEFAULT '',
  port       INTEGER NOT NULL DEFAULT 0 CHECK (port BETWEEN 0 AND 65535),
  user       TEXT NOT NULL DEFAULT '',
  password   TEXT NOT NULL DEFAULT '',
  database   TEXT NOT NULL CHECK (length(database) > 0),
  params     TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(params)),
  created_at INTEGER NOT NULL CHECK (created_at > 0),
  created_by TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE TABLE permissions (
  connection  TEXT NOT NULL CHECK (length(connection) > 0),
  member      TEXT NOT NULL CHECK (length(member) > 0),
  member_type TEXT NOT NULL CHECK (member_type IN (%s)),
  level       TEXT NOT NULL CHECK (level IN (%s)),
  expires_at  INTEGER NOT NULL CHECK (expires_at > 0),
  created_by  TEXT NOT NULL,
  created_at  INTEGER NOT NULL CHECK (created_at > 0),
  updated_at  INTEGER NOT NULL CHECK (updated_at > 0),
  PRIMARY KEY (connection, member)
) STRICT;

CREATE TABLE users (
  subject          TEXT PRIMARY KEY CHECK (length(trim(subject)) > 0),
  email            TEXT NOT NULL DEFAULT '',
  username         TEXT NOT NULL DEFAULT '',
  password         TEXT NOT NULL DEFAULT '',
  groups           TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(groups)),
  role             TEXT NOT NULL CHECK (role IN (%s)),
  status           TEXT NOT NULL CHECK (status IN (%s)),
  token_hash       TEXT NOT NULL DEFAULT '',
  token_expires_at INTEGER NOT NULL DEFAULT 0 CHECK (token_expires_at >= 0),
  last_login_at    INTEGER NOT NULL DEFAULT 0 CHECK (last_login_at >= 0),
  created_at       INTEGER NOT NULL CHECK (created_at > 0),
  updated_at       INTEGER NOT NULL CHECK (updated_at > 0)
) STRICT;

CREATE TABLE access_requests (
  id            TEXT PRIMARY KEY CHECK (length(id) > 0),
  subject       TEXT NOT NULL CHECK (length(subject) > 0),
  connection    TEXT NOT NULL CHECK (length(connection) > 0),
  level         TEXT NOT NULL CHECK (level IN (%s)),
  reason        TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL CHECK (status IN (%s)),
  requested_at  INTEGER NOT NULL CHECK (requested_at > 0),
  decided_by    TEXT NOT NULL DEFAULT '',
  decided_at    INTEGER NOT NULL DEFAULT 0 CHECK (decided_at >= 0),
  decision_note TEXT NOT NULL DEFAULT '',
  expires_at    INTEGER NOT NULL DEFAULT 0 CHECK (expires_at >= 0)
) STRICT;

CREATE TABLE access_events (
  id         TEXT PRIMARY KEY CHECK (length(id) > 0),
  at         INTEGER NOT NULL CHECK (at > 0),
  actor      TEXT NOT NULL,
  action     TEXT NOT NULL CHECK (length(action) > 0),
  subject    TEXT NOT NULL DEFAULT '',
  connection TEXT NOT NULL DEFAULT '',
  level      TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE UNIQUE INDEX connections_id ON connections(id);
CREATE UNIQUE INDEX users_email ON users(email) WHERE email != '';
CREATE UNIQUE INDEX users_username ON users(username) WHERE username != '';
CREATE UNIQUE INDEX users_token_hash ON users(token_hash) WHERE token_hash != '';
CREATE UNIQUE INDEX access_requests_pending ON access_requests(subject, connection) WHERE status = %s;
CREATE INDEX changes_by_created ON changes(created_at DESC, id);
CREATE INDEX changes_by_connection_created ON changes(connection, created_at DESC, id);
CREATE INDEX changes_by_schedule ON changes(status, scheduled_at) WHERE scheduled_at > 0;
CREATE INDEX permissions_by_member ON permissions(member, connection);
CREATE INDEX permissions_groups ON permissions(member_type, member);
CREATE INDEX access_requests_by_subject ON access_requests(subject, requested_at DESC);
CREATE INDEX access_requests_by_status_connection ON access_requests(status, connection, requested_at);
CREATE INDEX access_events_by_subject ON access_events(subject, at DESC);
CREATE INDEX access_events_by_actor ON access_events(actor, at DESC);
CREATE INDEX access_events_by_connection ON access_events(connection, at DESC);
`

func schemaSQL() string {
	return fmt.Sprintf(schemaTemplate,
		sqlValues(constants.KindDML, constants.KindDDL, constants.KindMongo),
		sqlValues(constants.ChangeValidated, constants.ChangeUnverified, constants.ChangeFailed, constants.ChangeApproved, constants.ChangeApplying, constants.ChangeApplied, constants.ChangeRejected),
		sqlValues(entity.MemberUser, entity.MemberGroup),
		sqlValues(entity.LevelRead, entity.LevelWrite, entity.LevelAdmin),
		sqlValues(entity.RoleSuper, entity.RoleAdmin, entity.RoleMember),
		sqlValues(entity.StatusActive, entity.StatusBanned),
		sqlValues(entity.LevelRead, entity.LevelWrite, entity.LevelAdmin),
		sqlValues(entity.RequestPending, entity.RequestApproved, entity.RequestRejected, entity.RequestCancelled, entity.RequestExpired),
		sqlValues(entity.RequestPending),
	)
}

func sqlValues(values ...string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return strings.Join(quoted, ", ")
}

// encryptPlaintextPasswords seals connection passwords saved while no secret key was configured.
func (s *Store) encryptPlaintextPasswords(ctx context.Context) error {
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

// queryRows runs a query and scans every row with scan (shared by the list methods, so the iteration lives in one place).
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
