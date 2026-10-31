// Package sqlite persists connections and changes (SQLite via modernc, cgo-free) — the app's only state.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
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

const schemaVersion = 2

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

// schemaTemplate is the DDL as SQL, so the one non-Go artifact in the repository reads like SQL. Its placeholders carry
// the CHECK lists that come from constants and entity instead of being retyped here.
//
//go:embed schema.sql
var schemaTemplate string

func schemaSQL() string {
	return fmt.Sprintf(schemaTemplate,
		sqlValues(constants.KindDML, constants.KindDDL, constants.KindMongo, constants.KindDocument, constants.KindSearch),
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
