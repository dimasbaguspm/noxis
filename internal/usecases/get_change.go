package usecases

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// SQLChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
type SQLChange struct {
	Changes interfaces.ChangeRepository
	Grants  interfaces.GrantRepository
}

// Serve runs it.
func (uc SQLChange) Serve(ctx context.Context, id string) (*entity.Change, error) {
	return sqlChange(ctx, uc.Changes, uc.Grants, id)
}
