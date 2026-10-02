package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"noxis/internal/entity"
)

func (r *SQLite) Report(ctx context.Context, s entity.NodeStatus) error {
	body := s.State
	if s.Detail != "" {
		body = s.State + ": " + s.Detail
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO status(node, body, ts) VALUES(?,?,?)
		ON CONFLICT(node) DO UPDATE SET body=excluded.body, ts=excluded.ts`,
		s.Node, body, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("report status: %w", err)
	}
	return nil
}

func (r *SQLite) List(ctx context.Context) ([]entity.NodeStatus, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT node, body, ts FROM status ORDER BY node`)
	if err != nil {
		return nil, fmt.Errorf("list status: %w", err)
	}
	defer rows.Close()
	var out []entity.NodeStatus
	for rows.Next() {
		var node, body string
		var ts int64
		if err := rows.Scan(&node, &body, &ts); err != nil {
			return nil, err
		}
		state, detail, found := strings.Cut(body, ": ")
		if !found {
			state, detail = body, ""
		}
		out = append(out, entity.NodeStatus{Node: node, State: state, Detail: detail, TS: ts})
	}
	return out, rows.Err()
}
