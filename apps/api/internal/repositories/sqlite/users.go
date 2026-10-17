package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// userColumns is the shared SELECT list, so every scan stays in step.
const userColumns = `subject, username, password, groups, is_admin, disabled`

// CountUsers reports how many accounts exist.
func (s *Store) CountUsers(ctx context.Context) (n int, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// GetUser returns an account by subject.
func (s *Store) GetUser(ctx context.Context, subject string) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE subject = ?`, subject))
}

// GetUserByUsername returns a local account by login name. OIDC reference rows
// have no username, so they never match.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE username = ? AND username != ''`, username))
}

// ListUsers returns every account, local names first, for the permission
// editor's member picker.
func (s *Store) ListUsers(ctx context.Context) (out []entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY username = '', username, subject`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var u entity.User
		var groups string
		if err := rows.Scan(&u.Subject, &u.Username, &u.Password, &groups, &u.Admin, &u.Disabled); err != nil {
			return nil, err
		}
		u.Groups = decodeGroups(groups)
		out = append(out, u)
	}
	return out, rows.Err()
}

// EnsureSubject records an OIDC identity the first time it is seen.
func (s *Store) EnsureSubject(ctx context.Context, subject string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if subject == "" {
		return errors.New("ensure subject: empty subject")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO users (subject, created_at, updated_at) VALUES (?, ?, ?) ON CONFLICT(subject) DO NOTHING`,
		subject, now, now); err != nil {
		return fmt.Errorf("ensure subject %s: %w", subject, err)
	}
	return nil
}

// SaveUser inserts or updates an account. The token columns are untouched on
// update, so saving an account does not log it out.
func (s *Store) SaveUser(ctx context.Context, u entity.User) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := u.Validate(); err != nil {
		return err
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO users (subject, username, password, groups, is_admin, disabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(subject) DO UPDATE SET
			username = excluded.username,
			password = excluded.password,
			groups = excluded.groups,
			is_admin = excluded.is_admin,
			disabled = excluded.disabled,
			updated_at = excluded.updated_at`,
		u.Subject, u.Username, u.Password, encodeGroups(u.Groups), u.Admin, u.Disabled, now, now); err != nil {
		return fmt.Errorf("save user %s: %w", u.Subject, err)
	}
	return nil
}

// SetUserPassword replaces the stored hash.
func (s *Store) SetUserPassword(ctx context.Context, subject, passwordHash string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET password = ?, updated_at = ? WHERE subject = ?`,
		passwordHash, time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("set password %s: %w", subject, err)
	}
	return nil
}

// SetUserToken makes tokenHash the account's single active token.
func (s *Store) SetUserToken(ctx context.Context, subject, tokenHash string, expiresAt int64) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET token_hash = ?, token_expires_at = ?, updated_at = ? WHERE subject = ?`,
		tokenHash, expiresAt, time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("set token %s: %w", subject, err)
	}
	return nil
}

// ClearUserToken logs the account out.
func (s *Store) ClearUserToken(ctx context.Context, subject string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET token_hash = '', token_expires_at = 0, updated_at = ? WHERE subject = ?`,
		time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("clear token %s: %w", subject, err)
	}
	return nil
}

// UserByToken returns the enabled account owning an unexpired token.
func (s *Store) UserByToken(ctx context.Context, tokenHash string, now int64) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if tokenHash == "" {
		return nil, constants.ErrNotFound
	}
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users
		 WHERE token_hash = ? AND token_expires_at > ? AND disabled = 0`, tokenHash, now))
}

// scanUser maps one row, translating no-rows into the repository sentinel.
func scanUser(row *sql.Row) (*entity.User, error) {
	u := &entity.User{}
	var groups string
	if err := row.Scan(&u.Subject, &u.Username, &u.Password, &groups, &u.Admin, &u.Disabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	u.Groups = decodeGroups(groups)
	return u, nil
}

func encodeGroups(groups []string) string {
	if groups == nil {
		return "[]"
	}
	b, _ := json.Marshal(groups)
	return string(b)
}

func decodeGroups(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
