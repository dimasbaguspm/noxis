package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// ChangeRepository persists proposed changes.
type ChangeRepository interface {
	ListChanges(ctx context.Context) ([]entity.Change, error)
	GetChange(ctx context.Context, id string) (*entity.Change, error)
	SaveChange(ctx context.Context, c entity.Change) error
	// ClaimChange atomically records the apply actor and moves one approved change to applying.
	ClaimChange(ctx context.Context, id string, at int64, actor string) (claimed bool, err error)
	DeleteChange(ctx context.Context, id string) error
}
