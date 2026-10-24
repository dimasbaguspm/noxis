package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// AccessEventRepository reads the append-only access audit trail.
type AccessEventRepository interface {
	ListAccessEvents(ctx context.Context, subject string, limit int) ([]entity.AccessEvent, error)
}
