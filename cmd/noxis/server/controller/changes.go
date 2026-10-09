package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/sql"
)

// The change routes are shared by SQL and MongoDB, each with its own queue:
// the {engine} path parameter decides which flow runs, and a memory cache
// serves no changes at all.

// submitChange validates a change and records it. Validation is the trust
// step: the query must be a write, and query + rollback are dry-run together
// in one rolled-back transaction — proving the rollback undoes the change.
// Nothing persists on submit.
func (d Deps) submitChange(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := d.raw(ctx, sq, ch.Connection, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
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
	if err := d.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// approve signs off a validated change; self-approval is fine here.
func (d Deps) approve(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := d.authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return *ch, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status, ch.ApprovedAt = constants.ChangeApproved, time.Now().Unix()
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// apply executes an approved change transactionally; a failure is recorded
// on the change, not just returned.
func (d Deps) apply(ctx context.Context, sq interfaces.SQLModule, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.Changes.GetChange(ctx, id)
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
	conn, err := d.raw(ctx, sq, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := sq.Apply(ctx, *conn, ch.Query)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = d.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// reject marks a change rejected — records are artifacts, never deleted, so
// the review trail survives.
func (d Deps) reject(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if err := d.authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		return entity.Change{}, err
	}
	if ch.Status == constants.ChangeApplied {
		return *ch, fmt.Errorf("change already applied — it cannot be rejected")
	}
	ch.Status, ch.RejectedAt = constants.ChangeRejected, time.Now().Unix()
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// sqlChanges lists only SQL changes (DML/DDL): the queue is per family.
func (d Deps) sqlChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	all, err := d.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := d.readableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if (all[i].Kind == constants.KindDML || all[i].Kind == constants.KindDDL) && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// sqlChange loads a change and insists it is a SQL one, so a MongoDB request
// cannot be opened through the SQL flow.
func (d Deps) sqlChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindDML && ch.Kind != constants.KindDDL {
		return nil, fmt.Errorf("change %s is %s, not a sql change", id, ch.Kind)
	}
	if err := d.authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Success 200 {object} object{changes=[]entity.Change}
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes [get]
func ListChanges(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			changes, err := d.sqlChanges(ctx)
			return map[string]any{"changes": changes}, err
		case interfaces.MongoModule:
			changes, err := d.mongoChanges(ctx)
			return map[string]any{"changes": changes}, err
		}
		return nil, unsupported(mod, "changes")
	})
}

// @Summary Submit a change (validates before recording)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes [post]
func SubmitChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			var ch entity.Change
			if err := decode(r, &ch); err != nil {
				return nil, err
			}
			return d.submitChange(ctx, m, ch)
		case interfaces.MongoModule:
			var body mongoChangeBody
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return d.submitMongoChange(ctx, m, body)
		}
		return nil, unsupported(mod, "changes")
	})
}

// @Summary Get one change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id} [get]
func GetChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			return d.sqlChange(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			return d.mongoChange(ctx, r.PathValue("id"))
		}
		return nil, unsupported(mod, "changes")
	})
}

// @Summary Approve a validated change (self-approval allowed)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id}/approve [post]
func ApproveChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.sqlChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.approve(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := d.mongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.approve(ctx, r.PathValue("id"))
		}
		return nil, unsupported(mod, "changes")
	})
}

// @Summary Apply an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id}/apply [post]
func ApplyChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.sqlChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.apply(ctx, m, r.PathValue("id"))
		case interfaces.MongoModule:
			return d.applyMongo(ctx, m, r.PathValue("id"))
		}
		return nil, unsupported(mod, "changes")
	})
}

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id}/reject [post]
func RejectChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.sqlChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.reject(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := d.mongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.reject(ctx, r.PathValue("id"))
		}
		return nil, unsupported(mod, "changes")
	})
}

// errUnauthorized maps to 401: the tick endpoint is key-guarded.
var errUnauthorized = errors.New("invalid or missing api key")

// TickResult is one scheduler pass: the ids it applied and the ids it failed
// to apply (the failure is already recorded on the change itself).
//
// @Description applied and failed change ids from one tick.
type TickResult struct {
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
}

// dueChange reports whether an approved change is scheduled and its slot has
// arrived. An immediate change (no schedule) is never due: the operator
// applies it by hand.
func dueChange(ch entity.Change, now int64) bool {
	return ch.Status == constants.ChangeApproved && ch.ScheduledAt > 0 && ch.ScheduledAt <= now
}

// Tick applies every due change through its own engine. The in-process
// scheduler and POST /internal/tick both call it, so a change applied at 14:00
// does not care who knocked.
func (d Deps) Tick(ctx context.Context) (res TickResult, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	all, err := d.Changes.ListChanges(ctx)
	if err != nil {
		return TickResult{}, err
	}
	now := time.Now().Unix()
	for i := range all {
		if !dueChange(all[i], now) {
			continue
		}
		if _, err := d.applyChange(ctx, all[i]); err != nil {
			res.Failed = append(res.Failed, all[i].ID)
			continue
		}
		res.Applied = append(res.Applied, all[i].ID)
	}
	return res, nil
}

// applyChange resolves a change's engine from its connection's driver and
// applies it through the matching family, so the tick is family-agnostic.
func (d Deps) applyChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	conn, err := d.Connections.Get(ctx, ch.Connection)
	if err != nil {
		return ch, err
	}
	mod, ok := d.moduleByDriver(conn.Driver)
	if !ok {
		return ch, fmt.Errorf("no engine serves driver %q", conn.Driver)
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		return d.apply(ctx, m, ch.ID)
	case interfaces.MongoModule:
		return d.applyMongo(ctx, m, ch.ID)
	}
	return ch, fmt.Errorf("driver %q cannot be applied", conn.Driver)
}

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Param X-API-Key header string true "NOXIS_TICK_KEY"
// @Success 200 {object} TickResult
// @Failure 401 {object} map[string]string
// @Router /internal/tick [post]
func PostTick(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if d.TickKey == "" {
			return nil, fmt.Errorf("%w: scheduling is not configured (set %s)", interfaces.ErrUnsupported, entity.EnvTickKey)
		}
		if r.Header.Get("X-API-Key") != d.TickKey {
			return nil, errUnauthorized
		}
		return d.Tick(ctx)
	})
}
