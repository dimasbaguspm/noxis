package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const changeColumns = `id, connection, notes, query, rollback, kind, status, checks, created_at, created_by,
	approved_at, applying_at, applied_at, rejected_at, est_duration_ms, est_affected, applied_affected,
	applied_duration_ms, scheduled_at, affected_tables, affected_columns`

// SaveChange upserts a change.
func (s *Store) SaveChange(ctx context.Context, c entity.Change) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	checks, err := json.Marshal(c.Checks)
	if err != nil {
		return fmt.Errorf("encode checks: %w", err)
	}
	tables, err := json.Marshal(c.AffectedTables)
	if err != nil {
		return fmt.Errorf("encode tables: %w", err)
	}
	columns, err := json.Marshal(c.AffectedColumns)
	if err != nil {
		return fmt.Errorf("encode columns: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO changes (`+changeColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connection=excluded.connection, notes=excluded.notes, query=excluded.query,
			rollback=excluded.rollback, kind=excluded.kind, status=excluded.status, checks=excluded.checks,
			created_by=excluded.created_by,
			approved_at=excluded.approved_at, applying_at=excluded.applying_at, applied_at=excluded.applied_at,
			rejected_at=excluded.rejected_at,
			est_duration_ms=excluded.est_duration_ms, est_affected=excluded.est_affected,
			applied_affected=excluded.applied_affected, applied_duration_ms=excluded.applied_duration_ms,
			scheduled_at=excluded.scheduled_at,
			affected_tables=excluded.affected_tables, affected_columns=excluded.affected_columns
		WHERE changes.status != ? OR excluded.status IN (?, ?)`,
		c.ID, c.Connection, c.Notes, c.Query, c.Rollback, c.Kind, c.Status, string(checks),
		c.CreatedAt, c.CreatedBy, c.ApprovedAt, c.ApplyingAt, c.AppliedAt, c.RejectedAt,
		c.EstDurationMs, c.EstAffected, c.AppliedAffected, c.AppliedDurationMs,
		c.ScheduledAt, string(tables), string(columns), constants.ChangeApplying, constants.ChangeFailed, constants.ChangeApplied)
	if err != nil {
		return fmt.Errorf("save change %s: %w", c.ID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("save change %s: rows affected: %w", c.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("save change %s: %w", c.ID, constants.ErrConflict)
	}
	return nil
}

// ClaimChange atomically moves one approved change to applying.
func (s *Store) ClaimChange(ctx context.Context, id string, at int64) (claimed bool, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	result, err := s.db.ExecContext(ctx, `UPDATE changes SET status = ?, applying_at = ? WHERE id = ? AND status = ?`,
		constants.ChangeApplying, at, id, constants.ChangeApproved)
	if err != nil {
		return false, fmt.Errorf("claim change %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim change %s: rows affected: %w", id, err)
	}
	return rows == 1, nil
}

// ListChanges returns changes, newest first.
func (s *Store) ListChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return queryRows(ctx, s.db, `SELECT `+changeColumns+` FROM changes ORDER BY created_at DESC, id`, scanChange)
}

// GetChange returns one change, or constants.ErrNotFound.
func (s *Store) GetChange(ctx context.Context, id string) (c *entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	row := s.db.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM changes WHERE id = ?`, id)
	ch, err := scanChange(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("change %s: %w", id, constants.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// DeleteChange removes a change (reject = delete).
func (s *Store) DeleteChange(ctx context.Context, id string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM changes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete change %s: %w", id, err)
	}
	return nil
}

func scanChange(row scanner) (entity.Change, error) {
	var (
		c       entity.Change
		checks  string
		tables  string
		columns string
	)
	if err := row.Scan(&c.ID, &c.Connection, &c.Notes, &c.Query, &c.Rollback, &c.Kind, &c.Status, &checks,
		&c.CreatedAt, &c.CreatedBy, &c.ApprovedAt, &c.ApplyingAt, &c.AppliedAt, &c.RejectedAt,
		&c.EstDurationMs, &c.EstAffected, &c.AppliedAffected, &c.AppliedDurationMs,
		&c.ScheduledAt, &tables, &columns); err != nil {
		return c, err
	}
	for _, target := range []struct {
		raw string
		dst *[]string
	}{{tables, &c.AffectedTables}, {columns, &c.AffectedColumns}} {
		if target.raw != "" {
			if err := json.Unmarshal([]byte(target.raw), target.dst); err != nil {
				return c, fmt.Errorf("decode affected identifiers for %s: %w", c.ID, err)
			}
		}
	}
	if checks != "" {
		if err := json.Unmarshal([]byte(checks), &c.Checks); err != nil {
			return c, fmt.Errorf("decode checks for %s: %w", c.ID, err)
		}
	}
	return c, nil
}
