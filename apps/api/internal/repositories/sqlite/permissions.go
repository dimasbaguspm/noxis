package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const permissionColumns = `connection, member, member_type, level, expires_at, created_by, created_at, updated_at`

// ListPermissions includes expired grants for display; authorization filters them.
func (s *Store) ListPermissions(ctx context.Context, connection string) (out []entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions WHERE connection = ? ORDER BY member`,
		connection)
	if err != nil {
		return nil, fmt.Errorf("list permissions %s: %w", connection, err)
	}
	defer rows.Close()
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.Connection, &p.Member, &p.MemberType, &p.Level,
			&p.ExpiresAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListAllPermissions returns every permission row, for the access overview.
func (s *Store) ListAllPermissions(ctx context.Context) (out []entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions ORDER BY connection, member`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.Connection, &p.Member, &p.MemberType, &p.Level,
			&p.ExpiresAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPermission upserts a grant and its audit event in one transaction.
func (s *Store) SetPermission(ctx context.Context, p entity.Permission) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := p.Validate(); err != nil {
		return err
	}
	now := time.Now().Unix()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = now
	}
	if p.ExpiresAt == 0 {
		p.ExpiresAt = now + int64(constants.AccessGrantTTL.Seconds())
	}
	if p.CreatedBy == "" {
		p.CreatedBy = "system"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("set permission %s/%s: %w", p.Connection, p.Member, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO permissions (`+permissionColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(connection, member) DO UPDATE SET
			member_type = excluded.member_type,
			level = excluded.level,
			expires_at = excluded.expires_at,
			created_by = excluded.created_by,
			updated_at = excluded.updated_at`,
		p.Connection, p.Member, p.MemberType, p.Level, p.ExpiresAt, p.CreatedBy, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("set permission %s/%s: %w", p.Connection, p.Member, err)
	}
	if err := recordEvent(ctx, tx, entity.AccessEvent{
		Action: constants.AccessGranted, Actor: p.CreatedBy,
		Subject: p.Member, Connection: p.Connection, Level: p.Level,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// ListPermissionsForMembers returns every permission held by one of members, across connections, ordered by connection then member.
func (s *Store) ListPermissionsForMembers(ctx context.Context, members []string) (out []entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if len(members) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(members)), ",")
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions WHERE member IN (`+placeholders+`) ORDER BY connection, member`, args...)
	if err != nil {
		return nil, fmt.Errorf("list permissions for members: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.Connection, &p.Member, &p.MemberType, &p.Level,
			&p.ExpiresAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListGroupNames returns the distinct groups that hold any permission.
func (s *Store) ListGroupNames(ctx context.Context) (out []string, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT member FROM permissions WHERE member_type = ? ORDER BY member`, entity.MemberGroup)
	if err != nil {
		return nil, fmt.Errorf("list permission groups: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, rows.Err()
}

// DeletePermission records revocation atomically; deleting an absent grant is a no-op.
func (s *Store) DeletePermission(ctx context.Context, connection, member, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete permission %s/%s: %w", connection, member, err)
	}
	defer func() { _ = tx.Rollback() }()
	var level string
	switch err := tx.QueryRowContext(ctx,
		`SELECT level FROM permissions WHERE connection = ? AND member = ?`, connection, member).Scan(&level); {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM permissions WHERE connection = ? AND member = ?`, connection, member); err != nil {
		return fmt.Errorf("delete permission %s/%s: %w", connection, member, err)
	}
	if err := recordEvent(ctx, tx, entity.AccessEvent{
		Action: constants.AccessRevoked, Actor: actor,
		Subject: member, Connection: connection, Level: level,
	}); err != nil {
		return err
	}
	return tx.Commit()
}
