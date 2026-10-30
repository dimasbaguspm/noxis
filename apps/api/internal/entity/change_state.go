package entity

import (
	"errors"
	"fmt"
	"time"

	"noxis/internal/constants"
)

// ErrChangeConflict marks a transition a concurrent claim already took, so the queue answers 409 instead of 400.
var ErrChangeConflict = errors.New("change is being applied")

// Approve signs off a validated change. Unverified is allowed on purpose: some drivers cannot dry-run DDL, and the reviewer approves with eyes open.
func (ch *Change) Approve(actor string, now int64) error {
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status, ch.ApprovedAt, ch.ApprovedBy = constants.ChangeApproved, now, actor
	return nil
}

// Reject closes a change without running it.
func (ch *Change) Reject(actor string, now int64) error {
	switch ch.Status {
	case constants.ChangeApplying:
		return ErrChangeConflict
	case constants.ChangeApplied:
		return errors.New("change already applied — it cannot be rejected")
	}
	ch.Status, ch.RejectedAt, ch.RejectedBy = constants.ChangeRejected, now, actor
	return nil
}

// Schedule sets (or, with 0, clears) an approved change's slot. A slot in the past would never be due, so it is refused rather than silently ignored.
func (ch *Change) Schedule(at int64, actor string, now int64) error {
	if ch.Status == constants.ChangeApplying {
		return ErrChangeConflict
	}
	if ch.Status != constants.ChangeApproved {
		return fmt.Errorf("change is %s, not approved — cannot schedule", ch.Status)
	}
	if at > 0 && at <= now {
		return errors.New("scheduled time must be in the future")
	}
	ch.ScheduledAt, ch.ScheduledBy = at, actor
	return nil
}

// Applyable reports whether a change may run now: approved, and not waiting for a scheduled slot.
func (ch *Change) Applyable(now int64) error {
	if ch.Status == constants.ChangeApplying {
		return ErrChangeConflict
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
func (ch *Change) Due(now int64) bool {
	return ch.Status == constants.ChangeApproved && ch.ScheduledAt > 0 && ch.ScheduledAt <= now
}

// MarkApplying records the claim its caller won.
func (ch *Change) MarkApplying(at int64, actor string) {
	ch.Status, ch.ApplyingAt, ch.AppliedBy = constants.ChangeApplying, at, actor
}

// MarkApplied records a successful run on the change with what it measured.
func (ch *Change) MarkApplied(res Result, now int64) {
	ch.Status, ch.AppliedAt = constants.ChangeApplied, now
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
}

// MarkFailed records a failed run on the change, where the operator will look for it.
func (ch *Change) MarkFailed(err error) {
	ch.Status = constants.ChangeFailed
	ch.Checks = append(ch.Checks, ChangeCheck{Target: "apply", Error: err.Error()})
}
