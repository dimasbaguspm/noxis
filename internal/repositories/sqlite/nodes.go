package sqlite

import (
	"context"
	"time"

	"noxis/internal/entity"
)

// Register records the node; re-register refreshes name/ip/timestamp.
func (s *Store) Register(ctx context.Context, id, name, ip string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO slaves(id, name, ip, registered_at) VALUES(?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, ip=excluded.ip, registered_at=excluded.registered_at`,
		id, name, ip, time.Now().Unix())
	return err
}

// List returns every registered slave.
func (s *Store) List(ctx context.Context) ([]entity.Slave, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, ip, registered_at FROM slaves ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.Slave
	for rows.Next() {
		var sl entity.Slave
		if err := rows.Scan(&sl.ID, &sl.Name, &sl.IP, &sl.RegisteredAt); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}
