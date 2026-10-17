package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// AccessEventRepository reads the append-only access audit trail. Events are
// written by the repository that changes access, in the same transaction.
type AccessEventRepository interface {
	ListAccessEvents(ctx context.Context, subject string, limit int) ([]entity.AccessEvent, error)
}
