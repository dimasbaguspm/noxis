package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Changes is the MongoDB change queue: a document write has no generic rollback, so it is validated against a scratch collection and reviewed, then applied for real.
type Changes struct {
	Store   interfaces.ChangeRepository
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.MongoChangeEngine
	Metrics *metrics.Metrics
}

// NewChanges wires the MongoDB change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.MongoChangeEngine, m *metrics.Metrics) *Changes {
	return &Changes{Store: store, Conns: conns, Engine: engine, Metrics: m}
}

// Driver is the driver whose queue this is.
func (changes *Changes) Driver() string { return constants.DriverMongo }

// Satisfier.
var _ interfaces.MongoChangeFlow = (*Changes)(nil)

// Submit validates a request and records it.
func (changes *Changes) Submit(ctx context.Context, conn entity.Connection, actor, notes string, req entity.MongoRequest) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	encoded, err := json.Marshal(req)
	if err != nil {
		return entity.Change{}, err
	}
	target := req.Database + "." + req.Collection
	ch := entity.Change{
		ID: idgen.UUID(), Connection: conn.Name, Notes: notes, Query: string(encoded),
		Kind: constants.KindMongo, Status: constants.ChangeValidated,
		CreatedAt: time.Now().Unix(), CreatedBy: actor, AffectedTables: []string{target},
	}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !changes.Engine.IsWrite(req.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in Explore", req.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, req.Op+" on "+target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (scratch collection)"}
	if res, err := changes.Engine.ValidateRequest(ctx, conn, req); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	if ch.Status == constants.ChangeFailed {
		logging.WarnWithCtx(ctx, "change rejected: "+dry.Error, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	} else {
		logging.InfoWithCtx(ctx, "change validated", "change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	}
	return changes.record(ctx, ch)
}

// Approve signs off a validated request.
func (changes *Changes) Approve(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = ch.Approve(actor, time.Now().Unix()); err != nil {
		return err
	}
	if err := changes.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change approved", "change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// Reject closes a request without running it.
func (changes *Changes) Reject(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = conflict(ch.Reject(actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := changes.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change rejected", "change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// Schedule sets or clears an approved request's slot.
func (changes *Changes) Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = conflict(ch.Schedule(at, actor, time.Now().Unix())); err != nil {
		return err
	}
	if err := changes.save(ctx, ch); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change schedule updated", "change", ch.ID, "connection", ch.Connection, "actor", actor, "scheduledAt", at)
	return nil
}

// Apply runs an approved request and records the outcome.
func (changes *Changes) Apply(ctx context.Context, ch *entity.Change, actor string) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	now := time.Now().Unix()
	if err := conflict(ch.Applyable(now)); err != nil {
		logging.WarnWithCtx(ctx, "change cannot be applied", "change", ch.ID, "reason", err.Error())
		return *ch, err
	}
	conn, err := changes.Conns.Get(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	var req entity.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	if err := changes.claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := changes.Engine.Apply(ctx, *conn, req)
	changes.count(ctx, err)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection, "actor", actor)
		ch.MarkFailed(err)
		if saveErr := changes.save(ctx, ch); saveErr != nil {
			return *ch, errors.Join(err, fmt.Errorf("save failed apply outcome for change %s: %w", ch.ID, saveErr))
		}
		return *ch, err
	}
	ch.MarkApplied(res, now)
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs, "actor", actor)
	if err := changes.save(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Tick applies every due scheduled MongoDB request.
func (changes *Changes) Tick(ctx context.Context, now int64, actor string) (out entity.TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	all, err := changes.Store.ListChanges(ctx)
	if err != nil {
		return entity.TickResult{}, err
	}
	res := entity.TickResult{}
	for i := range all {
		ch := all[i]
		if ch.Kind != constants.KindMongo || !ch.Due(now) {
			continue
		}
		if _, err := changes.Apply(ctx, &ch, actor); err != nil {
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

// claim atomically reserves an approved request before the engine write.
func (changes *Changes) claim(ctx context.Context, ch *entity.Change, at int64, actor string) error {
	claimed, err := changes.Store.ClaimChange(ctx, ch.ID, at, actor)
	if err != nil {
		return err
	}
	if !claimed {
		return constants.ErrConflict
	}
	ch.MarkApplying(at, actor)
	return nil
}

// record stores a validated request. A failed submission is returned unrecorded: a request that cannot run is never queued, but the caller still sees why.
func (changes *Changes) record(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := changes.Store.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// save persists a decided request.
func (changes *Changes) save(ctx context.Context, ch *entity.Change) error {
	return changes.Store.SaveChange(ctx, *ch)
}

// count records the outcome of an apply.
func (changes *Changes) count(ctx context.Context, err error) {
	if changes.Metrics == nil {
		return
	}
	if err != nil {
		changes.Metrics.ChangeFailed(ctx, changes.Driver(), "apply")
		return
	}
	changes.Metrics.ChangeApplied(ctx, changes.Driver())
}

// conflict maps the change's claim sentinel to the API's conflict, so a losing racer reads as a 409.
func conflict(err error) error {
	if errors.Is(err, entity.ErrChangeConflict) {
		return constants.ErrConflict
	}
	return err
}
