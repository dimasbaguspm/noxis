// Package changeflow holds state rules shared by the SQL and MongoDB change queues.
package changeflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Flow is the state side of a change queue: the rules of the artifact and the store that records it.
type Flow struct {
	Store interfaces.ChangeRepository
	// family is the engine family this flow serves (constants.FamilySQL).
	family string
}

// New wires the shared flow for one family.
func New(store interfaces.ChangeRepository, family string) Flow {
	return Flow{Store: store, family: family}
}

// Family is the engine family this flow serves.
func (flow Flow) Family() string { return flow.family }

// Approve signs off a validated change. Unverified is allowed on purpose: some drivers cannot dry-run DDL, and the reviewer approves with eyes open.
func (flow Flow) Approve(ctx context.Context, ch *entity.Change, actor string) error {
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status, ch.ApprovedAt, ch.ApprovedBy = constants.ChangeApproved, time.Now().Unix(), actor
	return flow.Save(ctx, ch)
}

// Reject closes a change without running it.
func (flow Flow) Reject(ctx context.Context, ch *entity.Change, actor string) error {
	if ch.Status == constants.ChangeApplying {
		return constants.ErrConflict
	}
	if ch.Status == constants.ChangeApplied {
		return fmt.Errorf("change already applied — it cannot be rejected")
	}
	ch.Status, ch.RejectedAt, ch.RejectedBy = constants.ChangeRejected, time.Now().Unix(), actor
	return flow.Save(ctx, ch)
}

// Schedule sets (or, with 0, clears) an approved change's slot. A slot in the past would never be due, so it is refused rather than silently ignored.
func (flow Flow) Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) error {
	if ch.Status == constants.ChangeApplying {
		return constants.ErrConflict
	}
	if ch.Status != constants.ChangeApproved {
		return fmt.Errorf("change is %s, not approved — cannot schedule", ch.Status)
	}
	if at > 0 && at <= time.Now().Unix() {
		return fmt.Errorf("scheduled time must be in the future")
	}
	ch.ScheduledAt, ch.ScheduledBy = at, actor
	return flow.Save(ctx, ch)
}

// Claim atomically reserves an approved change before an engine write.
func (flow Flow) Claim(ctx context.Context, ch *entity.Change, at int64, actor string) error {
	claimed, err := flow.Store.ClaimChange(ctx, ch.ID, at, actor)
	if err != nil {
		return err
	}
	if !claimed {
		return constants.ErrConflict
	}
	ch.Status, ch.ApplyingAt, ch.AppliedBy = constants.ChangeApplying, at, actor
	return nil
}

// Applyable reports whether a change may run now: approved, and not waiting for a scheduled slot.
func Applyable(ch *entity.Change, now int64) error {
	if ch.Status == constants.ChangeApplying {
		return constants.ErrConflict
	}
	if ch.Status != constants.ChangeApproved {
		return fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	if ch.ScheduledAt > now {
		return fmt.Errorf("change is scheduled for %s — the scheduler will apply it",
			time.Unix(ch.ScheduledAt, 0).Format(time.RFC3339))
	}
	return nil
}

// Due reports whether an approved change's scheduled slot has arrived. An immediate change (no schedule) is never due: the operator applies it by hand.
func Due(ch entity.Change, now int64) bool {
	return ch.Status == constants.ChangeApproved && ch.ScheduledAt > 0 && ch.ScheduledAt <= now
}

// Record stores a validated change. A failed submission is returned unrecorded: a change that cannot run is never queued, but the caller still sees why.
func (flow Flow) Record(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := flow.Store.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// Applied records a successful run on the change with what it measured.
func Applied(ch *entity.Change, res entity.Result, now int64) {
	ch.Status, ch.AppliedAt = constants.ChangeApplied, now
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
}

// Failed records a failed run on the change, where the operator will look for it.
func Failed(ch *entity.Change, err error) {
	ch.Status = constants.ChangeFailed
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
}

// Save persists a decided change.
func (flow Flow) Save(ctx context.Context, ch *entity.Change) error {
	return flow.Store.SaveChange(ctx, *ch)
}

// Tick applies every due change of this family through apply.
func Tick(ctx context.Context, store interfaces.ChangeRepository, now int64, mine func(entity.Change) bool, apply func(context.Context, *entity.Change) (entity.Change, error)) (entity.TickResult, error) {
	all, err := store.ListChanges(ctx)
	if err != nil {
		return entity.TickResult{}, err
	}
	res := entity.TickResult{}
	for i := range all {
		ch := all[i]
		if !mine(ch) || !Due(ch, now) {
			continue
		}
		if _, err := apply(ctx, &ch); err != nil {
			if errors.Is(err, constants.ErrConflict) {
				continue
			}
			res.Failed = append(res.Failed, ch.ID)
			continue
		}
		res.Applied = append(res.Applied, ch.ID)
	}
	return res, nil
}
