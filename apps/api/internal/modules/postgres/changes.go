package postgres

import (
	"context"
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

// Changes is the SQL change queue: a batch is validated against its own rollback, reviewed, then applied in a transaction.
type Changes struct {
	changeflow.Flow
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.SQLChangeEngine
	Metrics *metrics.Metrics
}

// NewChanges wires the SQL change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.SQLChangeEngine, m *metrics.Metrics) *Changes {
	return &Changes{
		Flow:    changeflow.New(store, constants.DriverPostgres),
		Conns:   conns,
		Engine:  engine,
		Metrics: m,
	}
}

// Driver is the driver whose queue this flow serves.
func (changes *Changes) Driver() string { return changes.Flow.Driver() }

// Satisfier.
var _ interfaces.SQLChangeFlow = (*Changes)(nil)

// Submit validates a batch and records it.
func (changes *Changes) Submit(ctx context.Context, conn entity.Connection, actor string, ch entity.Change) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	ch.ID, ch.CreatedAt, ch.CreatedBy = idgen.UUID(), time.Now().Unix(), actor
	ch.Kind, ch.Status = changes.Engine.Kind(ch.Query), constants.ChangeValidated
	ch.AffectedTables, ch.AffectedColumns = changes.Engine.Referenced(ch.Query)
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}
	if err := changes.Engine.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}
	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := changes.Engine.ValidateChange(ctx, conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
	case errors.Is(err, interfaces.ErrUnverifiable):
		ch.Status, dry.Error = constants.ChangeUnverified, err.Error()
	default:
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)
	switch ch.Status {
	case constants.ChangeFailed:
		logging.WarnWithCtx(ctx, "change rejected: "+dry.Error, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	case constants.ChangeUnverified:
		logging.WarnWithCtx(ctx, "change recorded unverified: the driver cannot dry-run it",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	default:
		logging.InfoWithCtx(ctx, "change validated",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "scratch", ch.ScratchSchema, "actor", actor)
	}
	return changes.Record(ctx, ch)
}

// Approve signs off a validated batch.
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

// Reject closes a batch without running it.
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

// Schedule sets or clears an approved batch's slot.
func (changes *Changes) Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = changes.Flow.Schedule(ctx, ch, at, actor); err != nil {
		return err
	}
	logging.InfoWithCtx(ctx, "change schedule updated", "change", ch.ID, "connection", ch.Connection, "actor", actor, "scheduledAt", at)
	return nil
}

// Apply runs an approved batch in a transaction and records the outcome.
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
	if err := changes.Claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := changes.Engine.Apply(ctx, *conn, ch.Query)
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

// Tick applies every due scheduled change on this driver's connections.
func (changes *Changes) Tick(ctx context.Context, now int64, actor string) (out entity.TickResult, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return changeflow.Tick(ctx, changes.Store, now, func(ch entity.Change) bool {
		return changes.serves(ctx, ch)
	}, func(ctx context.Context, ch *entity.Change) (entity.Change, error) {
		return changes.Apply(ctx, ch, actor)
	})
}

// serves reports whether this queue owns the change: a SQL change on one of its own driver's connections. Every driver runs its own queue, so a queue that claimed another driver's change would apply it through the wrong engine.
func (changes *Changes) serves(ctx context.Context, ch entity.Change) bool {
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return false
	}
	conn, err := changes.Conns.Get(ctx, ch.Connection)
	return err == nil && conn.Driver == changes.Driver()
}

// count records the outcome of an apply.
func (changes *Changes) count(ctx context.Context, driver string, err error) {
	if changes.Metrics == nil {
		return
	}
	if err != nil {
		changes.Metrics.ChangeFailed(ctx, driver, "apply")
		return
	}
	changes.Metrics.ChangeApplied(ctx, driver)
}
