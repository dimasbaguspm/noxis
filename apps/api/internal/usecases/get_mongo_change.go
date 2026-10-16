package usecases

import (
	"context"
	"fmt"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// MongoChange is the "MongoChange" use case.
type MongoChange struct {
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc MongoChange) Serve(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := uc.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	if err := Authorize(ctx, uc.Permissions, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}
