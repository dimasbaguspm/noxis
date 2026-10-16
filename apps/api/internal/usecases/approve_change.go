package usecases

import (
	"context"
	"fmt"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"time"
)

// Approve is the "Approve" use case.
type Approve struct {
	RegistryDeps
}

// Serve runs it.
func (uc Approve) Run(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ch, err := uc.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return *ch, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status, ch.ApprovedAt = constants.ChangeApproved, time.Now().Unix()
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}
