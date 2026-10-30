package cockroach

import (
	"context"
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

// Changes is the SQL change queue: a batch is validated against its own rollback, reviewed, then applied in a transaction.
type Changes struct {
	Store   interfaces.ChangeRepository
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.SQLChangeEngine
	Metrics *metrics.Metrics
}

// NewChanges wires the SQL change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.SQLChangeEngine, m *metrics.Metrics) *Changes {
	return &Changes{Store: store, Conns: conns, Engine: engine, Metrics: m}
}

// Driver is the driver whose queue this is.
func (changes *Changes) Driver() string { return constants.DriverCockroach }

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
	derived := changes.Engine.Kind(ch.Query)
	if err := ch.RequireKind(derived); err != nil {
		return entity.Change{}, err
	}
	ch.Kind, ch.Status = derived, constants.ChangeValidated
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
	return changes.record(ctx, ch)
}

// Approve signs off a validated batch.
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

// Reject closes a batch without running it.
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

// Schedule sets or clears an approved batch's slot.
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

// Apply runs an approved batch in a transaction and records the outcome.
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
	if err := changes.claim(ctx, ch, now, actor); err != nil {
		return *ch, err
	}
	res, err := changes.Engine.Apply(ctx, *conn, ch.Query)
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

// Tick applies every due scheduled change on this driver's connections.
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
		if !changes.serves(ctx, ch) || !ch.Due(now) {
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

// serves reports whether this queue owns the change: a SQL change on one of its own driver's connections. Every driver runs its own queue, so a queue that claimed another driver's change would apply it through the wrong engine.
func (changes *Changes) serves(ctx context.Context, ch entity.Change) bool {
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return false
	}
	conn, err := changes.Conns.Get(ctx, ch.Connection)
	return err == nil && conn.Driver == changes.Driver()
}

// claim atomically reserves an approved change before the engine write.
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

// record stores a validated change. A failed submission is returned unrecorded: a change that cannot run is never queued, but the caller still sees why.
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

// save persists a decided change.
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
