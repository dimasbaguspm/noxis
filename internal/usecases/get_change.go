package usecases

import (
	"context"
	"fmt"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// SQLChange is the "SQLChange" use case.
type SQLChange struct{ Base }

// Serve runs it.
func (uc SQLChange) Serve(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := uc.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return nil, fmt.Errorf("change %s is %s, not a sql change", id, ch.Kind)
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}
