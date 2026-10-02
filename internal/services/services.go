// Package services owns the external dependencies: the sqlite store (opened
// with a health check) and the gRPC transport connection. Bootstrap wires
// these into the app; nothing else constructs them.
package services

import (
	"context"
	"fmt"
	"time"

	"noxis/internal/repository"
)

// Services is the connection container currently holding the sqlite store.
type Services struct {
	DB *repository.SQLite
}

// OpenSQLite opens the master store and verifies it answers a ping.
func OpenSQLite(path string) (*Services, error) {
	db, err := repository.OpenSQLite(path)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return &Services{DB: db}, nil
}
