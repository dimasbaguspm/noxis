// Package queue holds the one implementation of a driver's change lifecycle: a change is validated by its engine, then
// recorded, reviewed, scheduled and applied. Every engine shares this path instead of copying it, so a fix to the state
// machine or the audit trail lands once rather than six times.
package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Applier runs an approved change's body against the connection that serves it: the engine owns what a body means, so a
// SQL batch and a document request travel the same lifecycle and differ only here.
type Applier func(ctx context.Context, conn entity.Connection, ch entity.Change) (entity.Result, error)

// Queue is one driver's lifecycle plus the change kinds it owns, so a queue never claims another engine's change.
type Queue struct {
	driver  string
	kinds   map[string]bool
	applier Applier
	store   interfaces.ChangeRepository
	conns   interfaces.ConnectionRepository
	metrics *metrics.Metrics
}

// New wires the lifecycle for one driver. kinds are the change kinds this queue owns and applier runs an approved body.
func New(driver string, kinds []string, conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, applier Applier, m *metrics.Metrics) *Queue {
	owned := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		owned[kind] = true
	}
	return &Queue{driver: driver, kinds: owned, applier: applier, store: store, conns: conns, metrics: m}
}

// Driver is the driver whose queue this is.
func (q *Queue) Driver() string { return q.driver }

// Satisfier.
var _ interfaces.ChangeFlow = (*Queue)(nil)

// Approve signs off a validated change.
func (q *Queue) Approve(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = ch.Approve(actor, time.Now().Unix()); err != nil {
		return err
	}
	if err := q.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change approved", "change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// Reject closes a change without running it.
func (q *Queue) Reject(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = conflict(ch.Reject(actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := q.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change rejected", "change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// Schedule sets or clears an approved change's slot.
func (q *Queue) Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = conflict(ch.Schedule(at, actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := q.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change schedule updated", "change", ch.ID, "connection", ch.Connection, "actor", actor, "scheduledAt", at)
	return nil
}

// Apply runs an approved change through its engine and records the outcome.
func (q *Queue) Apply(ctx context.Context, ch *entity.Change, actor string) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	now := time.Now().Unix()
	if err := conflict(ch.Applyable(now)); err != nil {
		logging.WarnWithCtx(ctx, "change cannot be applied", "change", ch.ID, "reason", err.Error())
		return *ch, err
	}
	conn, err := q.conns.Get(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	if err := q.claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := q.applier(ctx, *conn, *ch)
	q.count(ctx, err)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection, "actor", actor)
		ch.MarkFailed(err)
		if saveErr := q.save(ctx, ch); saveErr != nil {
			return *ch, errors.Join(err, fmt.Errorf("save failed apply outcome for change %s: %w", ch.ID, saveErr))
		}
		return *ch, err
	}
	ch.MarkApplied(res, now)
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs, "actor", actor)
	if err := q.save(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Tick applies every due scheduled change this queue owns.
func (q *Queue) Tick(ctx context.Context, now int64, actor string) (out entity.TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := q.store.ListChanges(ctx)
	if err != nil {
		return entity.TickResult{}, err
	}
	res := entity.TickResult{}
	for i := range all {
		ch := all[i]
		if !q.serves(ctx, ch) || !ch.Due(now) {
			continue
		}
		if _, err := q.Apply(ctx, &ch, actor); err != nil {
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

// Record stores a validated change. A failed submission is returned unrecorded: a change that cannot run is never
// queued, but the caller still sees why.
func (q *Queue) Record(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := q.store.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// serves reports whether this queue owns the change: a kind it serves, on one of its own driver's connections. Every
// driver runs its own queue, so a queue that claimed another driver's change would apply it through the wrong engine.
func (q *Queue) serves(ctx context.Context, ch entity.Change) bool {
	if !q.kinds[ch.Kind] {
		return false
	}
	conn, err := q.conns.Get(ctx, ch.Connection)
	return err == nil && conn.Driver == q.driver
}

// claim atomically reserves an approved change before the engine write.
func (q *Queue) claim(ctx context.Context, ch *entity.Change, at int64, actor string) error {
	claimed, err := q.store.ClaimChange(ctx, ch.ID, at, actor)
	if err != nil {
		return err
	}
	if !claimed {
		return constants.ErrConflict
	}
	ch.MarkApplying(at, actor)
	return nil
}

// save persists a decided change.
func (q *Queue) save(ctx context.Context, ch *entity.Change) error {
	return q.store.SaveChange(ctx, *ch)
}

// count records the outcome of an apply.
func (q *Queue) count(ctx context.Context, err error) {
	if q.metrics == nil {
		return
	}
	if err != nil {
		q.metrics.ChangeFailed(ctx, q.driver, "apply")
		return
	}
	q.metrics.ChangeApplied(ctx, q.driver)
}

// conflict maps the change's claim sentinel to the API's conflict, so a losing racer reads as a 409.
func conflict(err error) error {
	if errors.Is(err, entity.ErrChangeConflict) {
		return constants.ErrConflict
	}
	return err
}
