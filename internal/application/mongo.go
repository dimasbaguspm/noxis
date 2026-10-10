package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/mongodb"
)

// SubmitMongoChange validates a write request on a scratch collection and
// records it for review. Like the SQL flow, a request that fails validation is
// returned with its checks and never recorded.
func (a *App) SubmitMongoChange(ctx context.Context, m interfaces.MongoModule, conn, notes string, req interfaces.MongoRequest) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	raw, err := a.Raw(ctx, m, conn, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	ch := entity.Change{
		ID:         idgen.UUID(),
		Connection: conn,
		Notes:      notes,
		Kind:       constants.KindMongo,
		Status:     constants.ChangeValidated,
		CreatedAt:  time.Now().Unix(),
		CreatedBy:  actor(ctx),
		Rollback:   "",
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return entity.Change{}, err
	}
	// the request rides in Query as JSON: the console parses it back into the
	// filter and value editors.
	ch.Query = string(encoded)
	target := req.Database + "." + req.Collection
	ch.AffectedTables = []string{target}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !mongodb.IsWrite(req.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in Explore", req.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, req.Op+" on "+target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (scratch collection)"}
	if res, err := m.ValidateRequest(ctx, *raw, req); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := a.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// MongoChange loads a change and insists it is a MongoDB request, so the SQL
// queue cannot be addressed through the MongoDB flow.
func (a *App) MongoChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := a.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	if err := a.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// ApplyMongo executes an approved request; a failure is recorded on the
// change, and there is no rollback to fall back on.
func (a *App) ApplyMongo(ctx context.Context, m interfaces.MongoModule, id string) (out entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	ch, err := a.MongoChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := a.Raw(ctx, m, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	var req interfaces.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	res, err := m.Apply(ctx, *conn, req)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = a.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := a.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// MongoChanges lists only MongoDB requests: each family keeps its own queue.
func (a *App) MongoChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	all, err := a.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := a.readableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == constants.KindMongo && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}