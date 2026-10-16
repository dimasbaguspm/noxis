package usecases

import (
	"context"

	"noxis/internal/entity"
)

// SQLChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
type SQLChange struct {
	RegistryDeps
}

// Serve runs it.
func (uc SQLChange) Run(ctx context.Context, id string) (*entity.Change, error) {
	return sqlChange(ctx, uc.Changes, uc.Permissions, id)
}
