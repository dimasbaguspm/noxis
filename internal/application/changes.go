package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/sql"
)

// SubmitChange validates a change and records it. Validation is the trust
// step: the query must be a write, and query + rollback are dry-run together
// in one rolled-back transaction — proving the rollback undoes the change.
// Nothing persists on submit.
func (a *App) SubmitChange(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := a.Raw(ctx, sq, ch.Connection, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.CreatedBy = actor(ctx)
	ch.Kind = sq.Kind(ch.Query)
	ch.AffectedTables, ch.AffectedColumns = sql.Referenced(ch.Query)
	ch.Status = constants.ChangeValidated
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}

	if err := sq.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}

	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := sq.ValidateChange(ctx, *conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
	case errors.Is(err, sql.ErrUnverifiable):
		ch.Status, dry.Error = constants.ChangeUnverified, err.Error()
	default:
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	// a change that fails validation is a rejected proposal, not an artifact:
	// it is returned with its checks so the submitter sees why, and never
	// recorded.
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := a.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// Approve signs off a validated change; self-approval is fine here.
func (a *App) Approve(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := a.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return *ch, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status, ch.ApprovedAt = constants.ChangeApproved, time.Now().Unix()
	if err := a.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Schedule sets (or clears) the slot an approved change should run at. Only an
// approved change can be scheduled; 0 clears the schedule (apply by hand).
func (a *App) Schedule(ctx context.Context, id string, at int64) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := a.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot schedule", ch.Status)
	}
	if at > 0 && at <= time.Now().Unix() {
		return *ch, errors.New("scheduled time must be in the future")
	}
	ch.ScheduledAt = at
	if err := a.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Apply executes an approved change transactionally; a failure is recorded
// on the change, not just returned.
func (a *App) Apply(ctx context.Context, sq interfaces.SQLModule, id string) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	if ch.ScheduledAt > time.Now().Unix() {
		return *ch, fmt.Errorf("change is scheduled for %s — the scheduler will apply it",
			time.Unix(ch.ScheduledAt, 0).Format(time.RFC3339))
	}
	conn, err := a.Raw(ctx, sq, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := sq.Apply(ctx, *conn, ch.Query)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = a.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := a.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Reject marks a change rejected — records are artifacts, never deleted, so
// the review trail survives.
func (a *App) Reject(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := a.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status == constants.ChangeApplied {
		return *ch, fmt.Errorf("change already applied — it cannot be rejected")
	}
	ch.Status, ch.RejectedAt = constants.ChangeRejected, time.Now().Unix()
	if err := a.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// SQLChanges lists only SQL changes (DML/DDL): the queue is per family.
func (a *App) SQLChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	all, err := a.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := a.readableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if (all[i].Kind == constants.KindDML || all[i].Kind == constants.KindDDL) && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// SQLChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
func (a *App) SQLChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return nil, fmt.Errorf("change %s is %s, not a sql change", id, ch.Kind)
	}
	if err := a.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

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

// Tick applies every due change through its own engine. The in-process
// scheduler and POST /internal/tick both call it, so a change applied at 14:00
// does not care who knocked.
func (a *App) Tick(ctx context.Context) (res TickResult, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	all, err := a.Changes.ListChanges(ctx)
	if err != nil {
		return TickResult{}, err
	}
	now := time.Now().Unix()
	for i := range all {
		if !dueChange(all[i], now) {
			continue
		}
		if _, err := a.applyChange(ctx, all[i]); err != nil {
			res.Failed = append(res.Failed, all[i].ID)
			continue
		}
		res.Applied = append(res.Applied, all[i].ID)
	}
	return res, nil
}

// applyChange resolves a change's engine from its connection's driver and
// applies it through the matching family, so the tick is family-agnostic.
func (a *App) applyChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	conn, err := a.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return ch, err
	}
	mod, ok := a.moduleByDriver(conn.Driver)
	if !ok {
		return ch, fmt.Errorf("no engine serves driver %q", conn.Driver)
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		return a.Apply(ctx, m, ch.ID)
	case interfaces.MongoModule:
		return a.ApplyMongo(ctx, m, ch.ID)
	}
	return ch, fmt.Errorf("driver %q cannot be applied", conn.Driver)
}