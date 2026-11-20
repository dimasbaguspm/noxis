package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"noxis/internal/domain"
)

// Append writes journal lines in one transaction, so a change and the lines that describe
// it never disagree about how many of them exist.
func (d *DB) Append(ctx context.Context, entries ...domain.JournalEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: journal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, entry := range entries {
		detail := "{}"
		if len(entry.Detail) > 0 {
			buf, err := json.Marshal(entry.Detail)
			if err != nil {
				return fmt.Errorf("state: encode journal detail: %w", err)
			}
			detail = string(buf)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO journal (ref, subject_ref, subject_kind, kind, actor, detail, created_at)
			VALUES (?,?,?,?,?,?,?)`,
			entry.Ref, entry.SubjectRef, entry.SubjectKind, entry.Kind, entry.Actor, detail, millis(entry.CreatedAt),
		); err != nil {
			return fmt.Errorf("state: append journal: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: journal: %w", err)
	}
	return nil
}

// Entries answers the newest lines about one subject.
func (d *DB) Entries(ctx context.Context, subjectRef string, limit int) ([]domain.JournalEntry, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT ref, subject_ref, subject_kind, kind, actor, detail, created_at
		FROM journal WHERE subject_ref = ? ORDER BY ref DESC LIMIT ?`, subjectRef, limit)
	if err != nil {
		return nil, fmt.Errorf("state: list journal: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]domain.JournalEntry, 0, limit)
	for rows.Next() {
		var (
			entry     domain.JournalEntry
			detail    string
			createdAt int64
		)
		if err := rows.Scan(&entry.Ref, &entry.SubjectRef, &entry.SubjectKind, &entry.Kind,
			&entry.Actor, &detail, &createdAt); err != nil {
			return nil, fmt.Errorf("state: list journal: %w", err)
		}
		if err := json.Unmarshal([]byte(detail), &entry.Detail); err != nil {
			return nil, fmt.Errorf("state: decode journal detail: %w", err)
		}
		entry.CreatedAt = fromMillis(createdAt)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: list journal: %w", err)
	}
	return entries, nil
}
