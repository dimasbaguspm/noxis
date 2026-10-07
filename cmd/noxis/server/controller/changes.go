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
	conn, err := d.raw(ctx, sq, ch.Connection)
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
	conn, err := d.raw(ctx, sq, ch.Connection)
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
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == constants.KindDML || all[i].Kind == constants.KindDDL {
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
	return ch, nil
}

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Success 200 {object} object{changes=[]entity.Change}
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/changes [get]
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
// @Router /v1/{engine}/changes [post]
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
// @Router /v1/{engine}/changes/{id} [get]
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
// @Router /v1/{engine}/changes/{id}/approve [post]
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
// @Router /v1/{engine}/changes/{id}/apply [post]
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
// @Router /v1/{engine}/changes/{id}/reject [post]
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
