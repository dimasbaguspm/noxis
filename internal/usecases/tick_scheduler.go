package usecases

import (
	"context"
	"fmt"
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
// engine. The in-process scheduler and POST /internal/callback/scheduler both
// call it, so a change applied at 14:00 does not care who knocked.
type Tick struct{ Base }

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
		if _, err := uc.applyChange(ctx, all[i]); err != nil {
			res.Failed = append(res.Failed, all[i].ID)
			continue
		}
		res.Applied = append(res.Applied, all[i].ID)
	}
	return res, nil
}

// applyChange resolves a change's engine from its connection's driver and
// applies it through the matching family, so the tick is family-agnostic.
func (uc Tick) applyChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	conn, err := uc.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return ch, err
	}
	mod, ok := uc.moduleByDriver(conn.Driver)
	if !ok {
		return ch, fmt.Errorf("no engine serves driver %q", conn.Driver)
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		return (Apply(uc)).Serve(ctx, m, ch.ID)
	case interfaces.MongoModule:
		return (ApplyMongo(uc)).Serve(ctx, m, ch.ID)
	}
	return ch, fmt.Errorf("driver %q cannot be applied", conn.Driver)
}
