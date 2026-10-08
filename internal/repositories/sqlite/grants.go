package sqlite

import (
	"context"
	"fmt"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// ListGrants returns a connection's grants, ordered by group.
func (s *Store) ListGrants(ctx context.Context, connection string) (out []entity.Grant, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT connection, group_name, level FROM connection_grants WHERE connection = ? ORDER BY group_name`,
		connection)
	if err != nil {
		return nil, fmt.Errorf("list grants %s: %w", connection, err)
	}
	defer rows.Close()
	for rows.Next() {
		var g entity.Grant
		if err := rows.Scan(&g.Connection, &g.Group, &g.Level); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetGrant upserts one group's level for a connection.
func (s *Store) SetGrant(ctx context.Context, g entity.Grant) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := g.Validate(); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO connection_grants (connection, group_name, level) VALUES (?, ?, ?)
		ON CONFLICT(connection, group_name) DO UPDATE SET level = excluded.level`,
		g.Connection, g.Group, g.Level); err != nil {
		return fmt.Errorf("set grant %s/%s: %w", g.Connection, g.Group, err)
	}
	return nil
}

// DeleteGrant removes one group's grant from a connection.
func (s *Store) DeleteGrant(ctx context.Context, connection, group string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM connection_grants WHERE connection = ? AND group_name = ?`,
		connection, group); err != nil {
		return fmt.Errorf("delete grant %s/%s: %w", connection, group, err)
	}
	return nil
}
