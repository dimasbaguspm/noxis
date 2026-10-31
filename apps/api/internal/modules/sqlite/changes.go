package sqlite

import (
	"context"
	"errors"
	"time"

	"noxis/internal/changeflow"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// sqlKinds are the change kinds a SQL engine's queue owns: one driver never claims another engine's change.
var sqlKinds = []string{constants.KindDML, constants.KindDDL}

// Changes is this driver's SQL queue: its engine over the shared lifecycle.
type Changes struct {
	*changeflow.Queue
	Engine interfaces.SQLChangeEngine
}

// NewChanges wires the SQL change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.SQLChangeEngine, m *metrics.Metrics) *Changes {
	changes := &Changes{Engine: engine}
	changes.Queue = changeflow.New(constants.DriverSQLite, sqlKinds, conns, store, changes.apply, m)
	return changes
}

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
		ch.UnverifiedReason = interfaces.UnverifiableReason(err)
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

// apply runs an approved batch through this driver's engine.
func (changes *Changes) apply(ctx context.Context, conn entity.Connection, ch entity.Change) (Result, error) {
	return changes.Engine.Apply(ctx, conn, ch.Query)
}
