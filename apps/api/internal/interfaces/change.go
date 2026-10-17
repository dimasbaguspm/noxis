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
	DeleteChange(ctx context.Context, id string) error
}
