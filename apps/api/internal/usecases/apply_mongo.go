package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"time"
)

// ApplyMongo is the "ApplyMongo" use case.
type ApplyMongo struct {
	RegistryDeps
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc ApplyMongo) Run(ctx context.Context, m interfaces.MongoModule, id string) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ch, err := mongoChange(ctx, uc.Changes, uc.Permissions, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := rawConn(ctx, uc.Connections, uc.Permissions, uc.Modules, m, ch.Connection, entity.LevelAdmin)
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
		_ = uc.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}
