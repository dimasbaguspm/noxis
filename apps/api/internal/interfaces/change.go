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
	// ClaimChange atomically moves one approved change to applying; claimed is false if another transition won.
	ClaimChange(ctx context.Context, id string, at int64) (claimed bool, err error)
	DeleteChange(ctx context.Context, id string) error
}
