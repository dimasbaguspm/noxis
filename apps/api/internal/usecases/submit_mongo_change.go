package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"time"
)

// SubmitMongoChange is the "SubmitMongoChange" use case.
type SubmitMongoChange struct {
	RegistryDeps
}

// Serve runs it.
func (uc SubmitMongoChange) Run(ctx context.Context, m interfaces.MongoModule, conn, notes string, req interfaces.MongoRequest) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	raw, err := uc.RawConn(ctx, m, conn, entity.LevelWrite)
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
	if err := uc.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}
