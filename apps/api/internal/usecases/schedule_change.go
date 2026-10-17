package usecases

import (
	"context"
	"errors"
	"fmt"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"time"
)

// Schedule is the "Schedule" use case.
type Schedule struct {
	RegistryDeps
}

// Serve runs it.
func (uc Schedule) Run(ctx context.Context, id string, at int64) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ch, err := uc.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := Authorize(ctx, uc.Permissions, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot schedule", ch.Status)
	}
	if at > 0 && at <= time.Now().Unix() {
		return *ch, errors.New("scheduled time must be in the future")
	}
	ch.ScheduledAt = at
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}
