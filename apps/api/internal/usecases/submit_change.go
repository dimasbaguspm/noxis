package usecases

import (
	"context"
	"errors"
	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/sql"
	"time"
)

// SubmitChange is the "SubmitChange" use case.
type SubmitChange struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc SubmitChange) Run(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (out entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := rawConn(ctx, uc.Connections, uc.Permissions, sq, ch.Connection, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.CreatedBy = actor(ctx)
	ch.Kind = sq.Kind(ch.Query)
	ch.AffectedTables, ch.AffectedColumns = sql.Referenced(ch.Query)
	ch.Status = constants.ChangeValidated
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}

	if err := sq.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}

	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := sq.ValidateChange(ctx, *conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
	case errors.Is(err, sql.ErrUnverifiable):
		ch.Status, dry.Error = constants.ChangeUnverified, err.Error()
	default:
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	// a change that fails validation is a rejected proposal, not an artifact:
	// it is returned with its checks so the submitter sees why, and never
	// recorded.
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := uc.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}
