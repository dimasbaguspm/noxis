// Package interfaces holds the seams between layers (one file per domain).
package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// ConnectionRepository persists the console's connections.
type ConnectionRepository interface {
	List(ctx context.Context) ([]entity.Connection, error)
	Get(ctx context.Context, name string) (*entity.Connection, error)
	GetByID(ctx context.Context, id string) (*entity.Connection, error)
	Save(ctx context.Context, c entity.Connection) error
	Delete(ctx context.Context, name string) error
}
