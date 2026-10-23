package sql

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

// Changes is the SQL change queue: a batch is validated against its own
// rollback, reviewed, then applied in a transaction. The shared lifecycle lives
// in changeflow; the engine step is this package's.
type Changes struct {
	changeflow.Flow
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.SQLModule
	Metrics *metrics.Metrics
}

// NewChanges wires the SQL change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.SQLModule, m *metrics.Metrics) *Changes {
	return &Changes{
		Flow:    changeflow.New(store, constants.FamilySQL),
		Conns:   conns,
		Engine:  engine,
		Metrics: m,
	}
}

// Family is the engine family this flow serves.
func (c *Changes) Family() string { return c.Flow.Family() }

// Satisfier.
var _ interfaces.SQLChangeFlow = (*Changes)(nil)

// Submit validates a batch and records it. The caller has already authorized
// write on the connection.
func (c *Changes) Submit(ctx context.Context, conn entity.Connection, actor string, ch entity.Change) (entity.Change, error) {
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	ch.ID, ch.CreatedAt, ch.CreatedBy = idgen.UUID(), time.Now().Unix(), actor
	ch.Kind, ch.Status = c.Engine.Kind(ch.Query), constants.ChangeValidated
	ch.AffectedTables, ch.AffectedColumns = c.Engine.Referenced(ch.Query)
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}
	if err := c.Engine.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}
	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := c.Engine.ValidateChange(ctx, conn, ch.Query, ch.Rollback)
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
		logging.WarnWithCtx(ctx, "change rejected: "+dry.Error, "connection", ch.Connection, "kind", ch.Kind)
	case constants.ChangeUnverified:
		logging.WarnWithCtx(ctx, "change recorded unverified: the driver cannot dry-run it",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind)
	default:
		logging.InfoWithCtx(ctx, "change validated",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "scratch", ch.ScratchSchema)
	}
	return c.Record(ctx, ch, dry.Error)
}

// Approve signs off a validated batch.
func (c *Changes) Approve(ctx context.Context, ch *entity.Change) error {
	return c.Flow.Approve(ctx, ch)
}

// Reject closes a batch without running it.
func (c *Changes) Reject(ctx context.Context, ch *entity.Change) error { return c.Flow.Reject(ctx, ch) }

// Schedule sets or clears an approved batch's slot.
func (c *Changes) Schedule(ctx context.Context, ch *entity.Change, at int64) error {
	return c.Flow.Schedule(ctx, ch, at)
}

// Apply runs an approved batch in a transaction and records the outcome.
func (c *Changes) Apply(ctx context.Context, ch *entity.Change) (entity.Change, error) {
	now := time.Now().Unix()
	if err := changeflow.Applyable(ch, now); err != nil {
		logging.WarnWithCtx(ctx, "change cannot be applied", "change", ch.ID, "reason", err.Error())
		return *ch, err
	}
	conn, err := c.Conns.Get(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	if err := c.Claim(ctx, ch, now); err != nil {
		return *ch, err
	}
	res, err := c.Engine.Apply(ctx, *conn, ch.Query)
	c.count(conn.Family, err)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection)
		changeflow.Failed(ch, err)
		if saveErr := c.Save(ctx, ch); saveErr != nil {
			return *ch, errors.Join(err, fmt.Errorf("save failed apply outcome for change %s: %w", ch.ID, saveErr))
		}
		return *ch, err
	}
	changeflow.Applied(ch, res, now)
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs)
	if err := c.Save(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// Tick applies every due scheduled SQL change.
func (c *Changes) Tick(ctx context.Context, now int64) (entity.TickResult, error) {
	return changeflow.Tick(ctx, c.Store, now, func(ch entity.Change) bool {
		return ch.Kind == constants.KindDML || ch.Kind == constants.KindDDL
	}, c.Apply)
}

// count records the outcome of an apply.
func (c *Changes) count(family string, err error) {
	if c.Metrics == nil {
		return
	}
	if err != nil {
		c.Metrics.ChangeFailed(context.Background(), family, "apply")
		return
	}
	c.Metrics.ChangeApplied(context.Background(), family)
}
