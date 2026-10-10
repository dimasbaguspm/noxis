package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// mongoChange loads a change and insists it is a MongoDB request, so the SQL
// queue cannot be addressed through the MongoDB flow.
func mongoChange(ctx context.Context, changes interfaces.ChangeRepository, grants interfaces.GrantRepository, id string) (*entity.Change, error) {
	ch, err := changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	if err := Authorize(ctx, grants, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// sqlChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
func sqlChange(ctx context.Context, changes interfaces.ChangeRepository, grants interfaces.GrantRepository, id string) (*entity.Change, error) {
	ch, err := changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return nil, fmt.Errorf("change %s is %s, not a sql change", id, ch.Kind)
	}
	if err := Authorize(ctx, grants, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// applySQL executes an approved change transactionally; a failure is recorded
// on the change, not just returned.
func applySQL(ctx context.Context, changes interfaces.ChangeRepository, conns interfaces.ConnectionRepository, grants interfaces.GrantRepository, modules map[string]interfaces.Module, sq interfaces.SQLModule, id string) (out entity.Change, err error) {
	ch, err := changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	if ch.ScheduledAt > time.Now().Unix() {
		return *ch, fmt.Errorf("change is scheduled for %s — the scheduler will apply it",
			time.Unix(ch.ScheduledAt, 0).Format(time.RFC3339))
	}
	conn, err := Raw(ctx, conns, grants, modules, sq, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := sq.Apply(ctx, *conn, ch.Query)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// applyMongo executes an approved request; a failure is recorded on the
// change, and there is no rollback to fall back on.
func applyMongo(ctx context.Context, changes interfaces.ChangeRepository, conns interfaces.ConnectionRepository, grants interfaces.GrantRepository, modules map[string]interfaces.Module, m interfaces.MongoModule, id string) (out entity.Change, err error) {
	ch, err := mongoChange(ctx, changes, grants, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := Raw(ctx, conns, grants, modules, m, ch.Connection, entity.LevelAdmin)
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
		_ = changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// applyChange resolves a change's engine from its connection's driver and
// applies it through the matching family, so the scheduler is family-agnostic.
func applyChange(ctx context.Context, changes interfaces.ChangeRepository, conns interfaces.ConnectionRepository, grants interfaces.GrantRepository, modules map[string]interfaces.Module, ch entity.Change) (entity.Change, error) {
	conn, err := conns.Get(ctx, ch.Connection)
	if err != nil {
		return ch, err
	}
	mod, ok := moduleByDriver(modules, conn.Driver)
	if !ok {
		return ch, fmt.Errorf("no engine serves driver %q", conn.Driver)
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		return applySQL(ctx, changes, conns, grants, modules, m, ch.ID)
	case interfaces.MongoModule:
		return applyMongo(ctx, changes, conns, grants, modules, m, ch.ID)
	}
	return ch, fmt.Errorf("driver %q cannot be applied", conn.Driver)
}
