// Package interfaces is where the codebase agrees on its seams (one file per
// service) and their data shapes. Implementations live in
// internal/repositories/* and assert satisfiability right next to their
// structs. Thin on purpose — swap-ability in tests, not a taxonomy.
package interfaces

import (
	"context"
	"errors"

	"noxis/internal/entity"
)

// ErrNotFound reports a missing row.
var ErrNotFound = errors.New("not found")

// ConnectionRepository persists the SQL connections the console manages.
type ConnectionRepository interface {
	List(ctx context.Context) ([]entity.Connection, error)
	Get(ctx context.Context, name string) (*entity.Connection, error)
	Save(ctx context.Context, c entity.Connection) error
	Delete(ctx context.Context, name string) error
}

// ChangeRepository persists proposed changes (query + rollback + notes).
// Method names carry the noun because one Store implements both seams.
type ChangeRepository interface {
	ListChanges(ctx context.Context) ([]entity.Change, error)
	GetChange(ctx context.Context, id string) (*entity.Change, error)
	SaveChange(ctx context.Context, c entity.Change) error
	DeleteChange(ctx context.Context, id string) error
}
