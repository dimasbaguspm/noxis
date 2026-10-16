package usecases

import (
	"context"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// TickResult is one scheduler pass: the ids it applied and the ids it failed
// to apply (the failure is already recorded on the change itself).
//
// @Description applied and failed change ids from one tick.
type TickResult struct {
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
}

// dueChange reports whether an approved change is scheduled and its slot has
// arrived. An immediate change (no schedule) is never due: the operator
// applies it by hand.
func dueChange(ch entity.Change, now int64) bool {
	return ch.Status == constants.ChangeApproved && ch.ScheduledAt > 0 && ch.ScheduledAt <= now
}

// Tick is the scheduler-pass use case: every due change runs through its own
// engine. The external scheduler trigger (POST /internal/callback/scheduler)
// calls it, so a change applied at 14:00 does not care who knocked.
type Tick struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc Tick) Serve(ctx context.Context) (res TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		return TickResult{}, err
	}
	now := time.Now().Unix()
	for i := range all {
		if !dueChange(all[i], now) {
			continue
		}
		if _, err := applyChange(ctx, uc.Changes, uc.Connections, uc.Grants, all[i]); err != nil {
			res.Failed = append(res.Failed, all[i].ID)
			continue
		}
		res.Applied = append(res.Applied, all[i].ID)
	}
	return res, nil
}
