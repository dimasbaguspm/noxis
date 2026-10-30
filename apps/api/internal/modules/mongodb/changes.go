package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/changeflow"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Changes is the MongoDB change queue: a document write has no generic rollback, so it is validated against a scratch collection and reviewed, then applied for real.
type Changes struct {
	changeflow.Flow
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.MongoModule
	Metrics *metrics.Metrics
}

// NewChanges wires the MongoDB change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.MongoModule, m *metrics.Metrics) *Changes {
	return &Changes{
		Flow:    changeflow.New(store, constants.DriverMongo),
		Conns:   conns,
		Engine:  engine,
		Metrics: m,
	}
}

// Driver is the driver whose queue this flow serves.
func (changes *Changes) Driver() string { return changes.Flow.Driver() }

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
	return changes.Record(ctx, ch)
}

// Approve signs off a validated request.
func (changes *Changes) Approve(ctx context.Context, ch *entity.Change, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	previous := ch.Status
	if err = changes.Flow.Approve(ctx, ch, actor); err != nil {
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
	if err = changes.Flow.Reject(ctx, ch, actor); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change rejected", "change", ch.ID, "connection", ch.Connection, "actor", actor, "from", previous, "to", ch.Status)
	return nil
}

// Schedule sets or clears an approved request's slot.
func (changes *Changes) Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = changes.Flow.Schedule(ctx, ch, at, actor); err != nil {
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
	if err := changeflow.Applyable(ch, now); err != nil {
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
	if err := changes.Claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := changes.Engine.Apply(ctx, *conn, req)
	changes.count(ctx, changes.Driver(), err)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection, "actor", actor)
		changeflow.Failed(ch, err)
		if saveErr := changes.Save(ctx, ch); saveErr != nil {
			return *ch, errors.Join(err, fmt.Errorf("save failed apply outcome for change %s: %w", ch.ID, saveErr))
		}
		return *ch, err
	}
	changeflow.Applied(ch, res, now)
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs, "actor", actor)
	if err := changes.Save(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Tick applies every due scheduled MongoDB request.
func (changes *Changes) Tick(ctx context.Context, now int64, actor string) (out entity.TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return changeflow.Tick(ctx, changes.Store, now, func(ch entity.Change) bool {
		return ch.Kind == constants.KindMongo
	}, func(ctx context.Context, ch *entity.Change) (entity.Change, error) {
		return changes.Apply(ctx, ch, actor)
	})
}

// count records the outcome of an apply.
func (changes *Changes) count(ctx context.Context, family string, err error) {
	if changes.Metrics == nil {
		return
	}
	if err != nil {
		changes.Metrics.ChangeFailed(ctx, family, "apply")
		return
	}
	changes.Metrics.ChangeApplied(ctx, family)
}
