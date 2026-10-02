package repository

import (
	"context"
	"time"
)

func (r *SQLite) Register(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO nodes(id, registered_at) VALUES(?,?)
		 ON CONFLICT(id) DO NOTHING`, id, time.Now().Unix())
	return err
}

func (r *SQLite) ListIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM nodes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
