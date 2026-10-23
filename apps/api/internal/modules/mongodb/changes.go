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

// Changes is the MongoDB change queue: a document write has no generic
// rollback, so it is validated against a scratch collection and reviewed, then
// applied for real. The shared lifecycle lives in changeflow.
type Changes struct {
	changeflow.Flow
	Conns   interfaces.ConnectionRepository
	Engine  interfaces.MongoModule
	Metrics *metrics.Metrics
}

// NewChanges wires the MongoDB change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.MongoModule, m *metrics.Metrics) *Changes {
	return &Changes{
		Flow:    changeflow.New(store, constants.FamilyMongo),
		Conns:   conns,
		Engine:  engine,
		Metrics: m,
	}
}

// Family is the engine family this flow serves.
func (c *Changes) Family() string { return c.Flow.Family() }

// Satisfier.
var _ interfaces.MongoChangeFlow = (*Changes)(nil)

// Submit validates a request and records it. The caller has already authorized
// write on the connection.
func (c *Changes) Submit(ctx context.Context, conn entity.Connection, actor, notes string, req entity.MongoRequest) (entity.Change, error) {
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
	if !c.Engine.IsWrite(req.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in Explore", req.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, req.Op+" on "+target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (scratch collection)"}
	if res, err := c.Engine.ValidateRequest(ctx, conn, req); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	if ch.Status == constants.ChangeFailed {
		logging.WarnWithCtx(ctx, "change rejected: "+dry.Error, "connection", ch.Connection, "kind", ch.Kind)
	} else {
		logging.InfoWithCtx(ctx, "change validated", "change", ch.ID, "connection", ch.Connection, "kind", ch.Kind)
	}
	return c.Record(ctx, ch, dry.Error)
}

// Approve signs off a validated request.
func (c *Changes) Approve(ctx context.Context, ch *entity.Change) error {
	return c.Flow.Approve(ctx, ch)
}

// Reject closes a request without running it.
func (c *Changes) Reject(ctx context.Context, ch *entity.Change) error { return c.Flow.Reject(ctx, ch) }

// Schedule sets or clears an approved request's slot.
func (c *Changes) Schedule(ctx context.Context, ch *entity.Change, at int64) error {
	return c.Flow.Schedule(ctx, ch, at)
}

// Apply runs an approved request and records the outcome.
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
	var req entity.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	if err := c.Claim(ctx, ch, now); err != nil {
		return *ch, err
	}
	res, err := c.Engine.Apply(ctx, *conn, req)
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

// Tick applies every due scheduled MongoDB request.
func (c *Changes) Tick(ctx context.Context, now int64) (entity.TickResult, error) {
	return changeflow.Tick(ctx, c.Store, now, func(ch entity.Change) bool {
		return ch.Kind == constants.KindMongo
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
