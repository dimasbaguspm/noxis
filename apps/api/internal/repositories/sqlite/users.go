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
const userColumns = `subject, email, username, password, groups, role, status, last_login_at, created_at`

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

// GetUserByEmail returns the account blended to a verified email. Only rows with an email can match, so a blank claim never joins two strangers.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if email == "" {
		return nil, constants.ErrNotFound
	}
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = ? AND email != ''`, email))
}

// GetUserByLogin returns a local account by the identifier its owner types to sign in: the email, or the username for an account created with one.
func (s *Store) GetUserByLogin(ctx context.Context, login string) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if login == "" {
		return nil, constants.ErrNotFound
	}
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE (email = ? OR (username = ? AND username != '')) AND password != ''`,
		login, login))
}

// ListUsers returns every account, local names first, for the permission editor's member picker.
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
		if err := rows.Scan(&u.Subject, &u.Email, &u.Username, &u.Password, &groups, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Groups = decodeGroups(groups)
		out = append(out, u)
	}
	return out, rows.Err()
}

// EnsureSubject records an OIDC identity the first time it is seen.
func (s *Store) EnsureSubject(ctx context.Context, subject, email string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if subject == "" {
		return errors.New("ensure subject: empty subject")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO users (subject, email, role, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(subject) DO NOTHING`,
		subject, email, entity.RoleMember, entity.StatusActive, now, now); err != nil {
		return fmt.Errorf("ensure subject %s: %w", subject, err)
	}
	return nil
}

// SaveUser inserts or updates an account. The token columns are untouched on update, so saving an account does not log it out.
func (s *Store) SaveUser(ctx context.Context, u entity.User) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := u.Validate(); err != nil {
		return err
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO users (subject, email, username, password, groups, role, status, last_login_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(subject) DO UPDATE SET
			email = excluded.email,
			username = excluded.username,
			password = excluded.password,
			groups = excluded.groups,
			status = excluded.status,
			updated_at = excluded.updated_at`,
		u.Subject, u.Email, u.Username, u.Password, encodeGroups(u.Groups), u.Role, u.Status, u.LastLoginAt, now, now); err != nil {
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

// UserByToken returns the active account owning an unexpired token.
func (s *Store) UserByToken(ctx context.Context, tokenHash string, now int64) (u *entity.User, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if tokenHash == "" {
		return nil, constants.ErrNotFound
	}
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users
		 WHERE token_hash = ? AND token_expires_at > ? AND status = ?`, tokenHash, now, entity.StatusActive))
}

// SetUserStatus bans or re-activates an account. A ban also drops the session, so the account stops working on its next request rather than on expiry.
func (s *Store) SetUserStatus(ctx context.Context, subject, status string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	switch status {
	case entity.StatusActive, entity.StatusBanned:
	default:
		return fmt.Errorf("set status %s: unknown status %q", subject, status)
	}
	clearToken := status == entity.StatusBanned
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET status = ?, token_hash = CASE WHEN ? THEN '' ELSE token_hash END,
		        token_expires_at = CASE WHEN ? THEN 0 ELSE token_expires_at END, updated_at = ?
		 WHERE subject = ?`,
		status, clearToken, clearToken, time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("set status %s: %w", subject, err)
	}
	return nil
}

// SetUserRole sets the global role.
func (s *Store) SetUserRole(ctx context.Context, subject, role string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	switch role {
	case entity.RoleSuper, entity.RoleAdmin, entity.RoleMember:
	default:
		return fmt.Errorf("set role %s: unknown role %q", subject, role)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = ? WHERE subject = ?`,
		role, time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("set role %s: %w", subject, err)
	}
	return nil
}

func (s *Store) SetLastLogin(ctx context.Context, subject string, at int64) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ? WHERE subject = ?`, at, subject); err != nil {
		return fmt.Errorf("set last login %s: %w", subject, err)
	}
	return nil
}

// SetSoleSuper makes subject the only super admin.
func (s *Store) SetSoleSuper(ctx context.Context, subject string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sole super: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = ? WHERE role = ? AND subject != ?`,
		entity.RoleMember, time.Now().Unix(), entity.RoleSuper, subject); err != nil {
		return fmt.Errorf("clear super: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = ? WHERE subject = ?`,
		entity.RoleSuper, time.Now().Unix(), subject); err != nil {
		return fmt.Errorf("set super %s: %w", subject, err)
	}
	return tx.Commit()
}

// scanUser maps one row, translating no-rows into the repository sentinel.
func scanUser(row *sql.Row) (*entity.User, error) {
	u := &entity.User{}
	var groups string
	if err := row.Scan(&u.Subject, &u.Email, &u.Username, &u.Password, &groups, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt); err != nil {
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
