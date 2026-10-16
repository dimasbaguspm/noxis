package sqlite

import (
	"context"
	"fmt"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// ListPermissions returns a connection's permissions, ordered by member.
func (s *Store) ListPermissions(ctx context.Context, connection string) (out []entity.Permission, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT connection, member, member_type, level FROM permissions WHERE connection = ? ORDER BY member`,
		connection)
	if err != nil {
		return nil, fmt.Errorf("list permissions %s: %w", connection, err)
	}
	defer rows.Close()
	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.Connection, &p.Member, &p.MemberType, &p.Level); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPermission upserts one member's level for a connection.
func (s *Store) SetPermission(ctx context.Context, p entity.Permission) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := p.Validate(); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO permissions (connection, member, member_type, level) VALUES (?, ?, ?, ?)
		ON CONFLICT(connection, member) DO UPDATE SET member_type = excluded.member_type, level = excluded.level`,
		p.Connection, p.Member, p.MemberType, p.Level); err != nil {
		return fmt.Errorf("set permission %s/%s: %w", p.Connection, p.Member, err)
	}
	return nil
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

// DeletePermission removes one member's permission from a connection.
func (s *Store) DeletePermission(ctx context.Context, connection, member string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM permissions WHERE connection = ? AND member = ?`,
		connection, member); err != nil {
		return fmt.Errorf("delete permission %s/%s: %w", connection, member, err)
	}
	return nil
}
