package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"noxis/internal/entity"
)

func (r *SQLite) Get(ctx context.Context, node string) (*entity.Spec, error) {
	var body string
	err := r.db.QueryRowContext(ctx, `SELECT body FROM specs WHERE node=?`, node).Scan(&body)
	if err == sql.ErrNoRows {
		return &entity.Spec{Node: node}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get spec %s: %w", node, err)
	}
	var stored entity.StoredSpec
	if err := yaml.Unmarshal([]byte(body), &stored); err != nil {
		return nil, fmt.Errorf("spec %s: %w", node, err)
	}
	return &entity.Spec{Node: node, Engines: stored.Engines}, nil
}

func (r *SQLite) Set(ctx context.Context, spec *entity.Spec) error {
	stored := entity.StoredSpec{Engines: spec.Engines}
	body, err := yaml.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO specs(node, body, updated_at) VALUES(?,?,?)
		ON CONFLICT(node) DO UPDATE SET body=excluded.body, updated_at=excluded.updated_at`,
		spec.Node, string(body), time.Now().Unix()); err != nil {
		return fmt.Errorf("write spec: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO changelog(node, event, at) VALUES(?,?,?)`,
		spec.Node, "set_desired_state", time.Now().Unix()); err != nil {
		return fmt.Errorf("changelog: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.Notify(ctx, spec.Node)
	return nil
}

func (r *SQLite) Subscribe(_ context.Context, node string) chan struct{} {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sub[node] == nil {
		r.sub[node] = map[chan struct{}]struct{}{}
	}
	r.sub[node][ch] = struct{}{}
	return ch
}

func (r *SQLite) Unsubscribe(_ context.Context, node string, ch chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m := r.sub[node]; m != nil {
		delete(m, ch)
	}
}
