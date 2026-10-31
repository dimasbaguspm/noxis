package splunk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"noxis/internal/changeflow"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// searchKinds are the change kinds this queue owns: a search-platform change, reviewed like any other.
var searchKinds = []string{constants.KindSearch}

// Changes is the search queue: this driver's engine over the shared lifecycle.
type Changes struct {
	*changeflow.Queue
	Engine interfaces.SearchEngine
}

// NewChanges wires the search change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.SearchEngine, m *metrics.Metrics) *Changes {
	changes := &Changes{Engine: engine}
	changes.Queue = changeflow.New(constants.DriverSplunk, searchKinds, conns, store, changes.apply, m)
	return changes
}

// Satisfier.
var _ interfaces.SearchChangeFlow = (*Changes)(nil)

// Submit proves a request and records it.
func (changes *Changes) Submit(ctx context.Context, conn entity.Connection, actor, notes string, req entity.SearchRequest) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	encoded, err := json.Marshal(req)
	if err != nil {
		return entity.Change{}, err
	}
	ch := entity.Change{
		ID: idgen.UUID(), Connection: conn.Name, Notes: notes, Query: string(encoded),
		Kind: constants.KindSearch, Status: constants.ChangeValidated,
		CreatedAt: time.Now().Unix(), CreatedBy: actor, AffectedTables: []string{req.Target},
	}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !changes.Engine.IsWrite(req.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in the search page", req.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, req.Op+" on "+req.Target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (impact count)"}
	if res, err := changes.Engine.ValidateChange(ctx, conn, req); err == nil {
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

// apply runs an approved request through the search engine.
func (changes *Changes) apply(ctx context.Context, conn entity.Connection, ch entity.Change) (entity.Result, error) {
	var req entity.SearchRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Result{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	return changes.Engine.Apply(ctx, conn, req)
}
