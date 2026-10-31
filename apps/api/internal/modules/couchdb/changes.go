package couchdb

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

// documentKinds are the change kinds this queue owns: a document write, reviewed like a MongoDB request.
var documentKinds = []string{constants.KindDocument}

// Changes is the document queue: this driver's engine over the shared lifecycle.
type Changes struct {
	*changeflow.Queue
	Engine interfaces.MongoChangeEngine
}

// NewChanges wires the document change queue.
func NewChanges(conns interfaces.ConnectionRepository, store interfaces.ChangeRepository, engine interfaces.MongoChangeEngine, m *metrics.Metrics) *Changes {
	changes := &Changes{Engine: engine}
	changes.Queue = changeflow.New(constants.DriverCouchDB, documentKinds, conns, store, changes.apply, m)
	return changes
}

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
		Kind: constants.KindDocument, Status: constants.ChangeValidated,
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

	dry := entity.ChangeCheck{Target: "dry-run (scratch database)"}
	if res, err := changes.Engine.ValidateRequest(ctx, conn, req); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
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

// apply runs an approved request through the document engine.
func (changes *Changes) apply(ctx context.Context, conn entity.Connection, ch entity.Change) (entity.Result, error) {
	var req entity.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Result{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	return changes.Engine.Apply(ctx, conn, req)
}
