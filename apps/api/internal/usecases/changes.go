package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The change lifecycle is orchestration, not engine behaviour: a module proves a body and runs it, and everything else —
// recording, reviewing, scheduling, claiming, auditing — is the same whatever the engine is. It lives here, on the
// framework every use case embeds, so no module depends on another module's idea of a change.

// ApproveChange signs off a validated change.
func (d RegistryDeps) ApproveChange(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = ch.Approve(actor, time.Now().Unix()); err != nil {
		return err
	}
	if err := d.Repositories.Changes.SaveChange(ctx, *ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change approved",
		"change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// RejectChange closes a change without running it.
func (d RegistryDeps) RejectChange(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = conflict(ch.Reject(actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := d.Repositories.Changes.SaveChange(ctx, *ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change rejected",
		"change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// ScheduleChange sets or clears an approved change's slot.
func (d RegistryDeps) ScheduleChange(ctx context.Context, ch *entity.Change, at int64, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = conflict(ch.Schedule(at, actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := d.Repositories.Changes.SaveChange(ctx, *ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change schedule updated",
		"change", ch.ID, "connection", ch.Connection, "actor", actor, "scheduledAt", at)
	return nil
}

// ApplyChange runs an approved change through its engine and records the outcome.
func (d RegistryDeps) ApplyChange(ctx context.Context, ch *entity.Change, actor string) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	now := time.Now().Unix()
	if err := conflict(ch.Applyable(now)); err != nil {
		logging.WarnWithCtx(ctx, "change cannot be applied", "change", ch.ID, "reason", err.Error())
		return *ch, err
	}
	conn, err := d.Repositories.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	if err := d.claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := d.applyBody(ctx, *conn, *ch)
	d.countApply(ctx, conn.Driver, err)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection, "actor", actor)
		ch.MarkFailed(err)
		if saveErr := d.Repositories.Changes.SaveChange(ctx, *ch); saveErr != nil {
			return *ch, errors.Join(err, fmt.Errorf("save failed apply outcome for change %s: %w", ch.ID, saveErr))
		}
		return *ch, err
	}
	ch.MarkApplied(res, now)
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs, "actor", actor)
	if err := d.Repositories.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// TickChanges applies every due scheduled change this process can run.
func (d RegistryDeps) TickChanges(ctx context.Context, now int64, actor string) (out entity.TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := d.Repositories.Changes.ListChanges(ctx)
	if err != nil {
		return entity.TickResult{}, err
	}
	res := entity.TickResult{}
	for i := range all {
		ch := all[i]
		if !ch.Due(now) || !d.serves(ctx, ch) {
			continue
		}
		if _, err := d.ApplyChange(ctx, &ch, actor); err != nil {
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

// RecordChange stores a validated change. A failed submission is returned unrecorded: a change that cannot run is never
// queued, but the caller still sees why.
func (d RegistryDeps) RecordChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := d.Repositories.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// applyBody runs the change's body through the engine that serves its connection: the kind picks the capability, so a
// document request never travels through a SQL engine.
func (d RegistryDeps) applyBody(ctx context.Context, conn entity.Connection, ch entity.Change) (entity.Result, error) {
	module, err := d.Modules.Engine(conn.Driver)
	if err != nil {
		return entity.Result{}, err
	}
	switch ch.Kind {
	case constants.KindDML, constants.KindDDL:
		engine, ok := module.(interfaces.SQLChangeEngine)
		if !ok {
			return entity.Result{}, unsupportedKind(module, ch.Kind)
		}
		return engine.Apply(ctx, conn, ch.Query)
	case constants.KindMongo, constants.KindDocument:
		engine, ok := module.(interfaces.MongoChangeEngine)
		if !ok {
			return entity.Result{}, unsupportedKind(module, ch.Kind)
		}
		var req entity.MongoRequest
		if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
			return entity.Result{}, fmt.Errorf("stored request is unreadable: %w", err)
		}
		return engine.Apply(ctx, conn, req)
	case constants.KindSearch:
		engine, ok := module.(interfaces.SearchEngine)
		if !ok {
			return entity.Result{}, unsupportedKind(module, ch.Kind)
		}
		var req entity.SearchRequest
		if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
			return entity.Result{}, fmt.Errorf("stored request is unreadable: %w", err)
		}
		return engine.Apply(ctx, conn, req)
	}
	return entity.Result{}, unsupportedKind(module, ch.Kind)
}

// serves reports whether the engine behind the change's connection can run this kind of change at all: a driver with no
// change capability has nothing to apply.
func (d RegistryDeps) serves(ctx context.Context, ch entity.Change) bool {
	conn, err := d.Repositories.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return false
	}
	module, err := d.Modules.Engine(conn.Driver)
	if err != nil {
		return false
	}
	switch ch.Kind {
	case constants.KindDML, constants.KindDDL:
		_, ok := module.(interfaces.SQLChangeEngine)
		return ok
	case constants.KindMongo, constants.KindDocument:
		_, ok := module.(interfaces.MongoChangeEngine)
		return ok
	case constants.KindSearch:
		_, ok := module.(interfaces.SearchEngine)
		return ok
	}
	return false
}

// claim atomically reserves an approved change before the engine write.
func (d RegistryDeps) claim(ctx context.Context, ch *entity.Change, at int64, actor string) error {
	claimed, err := d.Repositories.Changes.ClaimChange(ctx, ch.ID, at, actor)
	if err != nil {
		return err
	}
	if !claimed {
		return constants.ErrConflict
	}
	ch.MarkApplying(at, actor)
	return nil
}

// countApply records the outcome of an apply against the driver that ran it.
func (d RegistryDeps) countApply(ctx context.Context, driver string, err error) {
	if err != nil {
		d.Metrics.ChangeFailed(ctx, driver, "apply")
		return
	}
	d.Metrics.ChangeApplied(ctx, driver)
}

// unsupportedKind names the engine that cannot run a kind of change, so the refusal says which and why.
func unsupportedKind(module interfaces.Module, kind string) error {
	return fmt.Errorf("%w: engine %q does not apply %s changes", constants.ErrUnsupported, module.Driver(), kind)
}

// conflict maps the change's claim sentinel to the API's conflict, so a losing racer reads as a 409.
func conflict(err error) error {
	if errors.Is(err, entity.ErrChangeConflict) {
		return constants.ErrConflict
	}
	return err
}
