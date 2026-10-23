// Package apply holds the change-application use cases and the scheduler pass.
package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Apply applies an approved change.
type Apply struct{ usecases.RegistryDeps }

// @Summary Apply an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/apply [post]
func (uc Apply) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	id := r.PathValue("id")
	switch m := mod.(type) {
	case interfaces.SQLModule:
		if _, err := uc.SQLChange(ctx, id); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := applySQL(ctx, uc.RegistryDeps, m, id)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Respond(w, out)
	case interfaces.MongoModule:
		out, err := applyMongo(ctx, uc.RegistryDeps, m, id)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Respond(w, out)
	default:
		uc.Fail(ctx, w, fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, mod.Family()))
	}
}

// applySQL executes an approved change transactionally; a failure is recorded
// on the change, not just returned.
func applySQL(ctx context.Context, d usecases.RegistryDeps, sq interfaces.SQLModule, id string) (entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		logging.WarnWithCtx(ctx, "change cannot be applied: it is not approved",
			"change", ch.ID, "status", ch.Status)
		return *ch, fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	if ch.ScheduledAt > time.Now().Unix() {
		logging.InfoWithCtx(ctx, "change deferred to the scheduler",
			"change", ch.ID, "scheduledAt", ch.ScheduledAt)
		return *ch, fmt.Errorf("change is scheduled for %s — the scheduler will apply it",
			time.Unix(ch.ScheduledAt, 0).Format(time.RFC3339))
	}
	conn, err := d.RawConn(ctx, sq, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := sq.Apply(ctx, *conn, ch.Query)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection)
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = d.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs)
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// applyMongo executes an approved request; a failure is recorded on the
// change, and there is no rollback to fall back on.
func applyMongo(ctx context.Context, d usecases.RegistryDeps, m interfaces.MongoModule, id string) (entity.Change, error) {
	ch, err := d.MongoChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := d.RawConn(ctx, m, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	var req interfaces.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	res, err := m.Apply(ctx, *conn, req)
	if err != nil {
		logging.ErrorWithCtx(ctx, "change failed to apply", err, "change", ch.ID, "connection", ch.Connection)
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = d.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	logging.InfoWithCtx(ctx, "change applied",
		"change", ch.ID, "connection", ch.Connection, "affected", res.Affected, "durationMs", res.DurationMs)
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// applyChange resolves a change's engine from its connection's driver and
// applies it through the matching family, so the scheduler is family-agnostic.
// It is the one place a change is applied, so it is where the outcome is
// counted.
func applyChange(ctx context.Context, d usecases.RegistryDeps, ch entity.Change) (entity.Change, error) {
	conn, err := d.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return ch, err
	}
	mod, ok := d.Modules.Serves(conn.Driver)
	if !ok {
		return ch, fmt.Errorf("no engine serves driver %q", conn.Driver)
	}
	var out entity.Change
	switch m := mod.(type) {
	case interfaces.SQLModule:
		out, err = applySQL(ctx, d, m, ch.ID)
	case interfaces.MongoModule:
		out, err = applyMongo(ctx, d, m, ch.ID)
	default:
		return ch, fmt.Errorf("driver %q cannot be applied", conn.Driver)
	}
	if err != nil {
		d.Metrics.ChangeFailed(ctx, conn.Family, "apply")
		return out, err
	}
	d.Metrics.ChangeApplied(ctx, conn.Family)
	return out, nil
}

// dueChange reports whether an approved change is scheduled and its slot has
// arrived. An immediate change (no schedule) is never due: the operator
// applies it by hand.
func dueChange(ch entity.Change, now int64) bool {
	return ch.Status == constants.ChangeApproved && ch.ScheduledAt > 0 && ch.ScheduledAt <= now
}

// Tick applies every due scheduled change; the external scheduler drives it.
type Tick struct{ usecases.RegistryDeps }

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Success 200 {object} entity.TickResult
// @Router /internal/callback/scheduler [post]
func (uc Tick) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	res := entity.TickResult{}
	now := time.Now().Unix()
	for i := range all {
		if !dueChange(all[i], now) {
			continue
		}
		// one span per due change: a tick that applies five changes shows five
		// operations in the trace, not one
		changeCtx, end := logging.StartSpan(ctx, "Scheduler.ApplyChange")
		logging.Attach(changeCtx, "change", all[i].ID, "connection", all[i].Connection)
		_, err := applyChange(changeCtx, uc.RegistryDeps, all[i])
		end(err)
		if err != nil {
			res.Failed = append(res.Failed, all[i].ID)
			continue
		}
		res.Applied = append(res.Applied, all[i].ID)
	}
	status := "ok"
	if len(res.Failed) > 0 {
		status = "failed"
	}
	uc.Metrics.SchedulerTick(ctx, status)
	uc.Respond(w, res)
}
