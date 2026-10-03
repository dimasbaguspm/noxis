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
	"noxis/internal/services/sql"
)

// submitChange validates a change and records it. Validation is the trust
// step: the query must be a write, and query + rollback are dry-run together
// in one rolled-back transaction — proving the rollback undoes the change.
// Nothing persists on submit.
func (d Deps) submitChange(ctx context.Context, ch entity.Change) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := d.raw(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.Kind = d.SQL.Kind(ch.Query)
	ch.Status = constants.ChangeValidated
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}

	if err := d.SQL.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}

	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := d.SQL.ValidateChange(ctx, *conn, ch.Query, ch.Rollback)
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
func (d Deps) apply(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := d.raw(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := d.SQL.Apply(ctx, *conn, ch.Query)
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

// @Summary List proposed changes (newest first)
// @Success 200 {object} object{changes=[]entity.Change}
// @Router /v1/changes [get]
func ListChanges(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		changes, err := d.Changes.ListChanges(ctx)
		return map[string]any{"changes": changes}, err
	})
}

// @Summary Submit a change (validates query + rollback; nothing persists)
// @Param body body entity.Change true "connection, notes, query, rollback"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes [post]
func SubmitChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var ch entity.Change
		if err := decode(r, &ch); err != nil {
			return nil, err
		}
		return d.submitChange(ctx, ch)
	})
}

// @Summary Get one change
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/changes/{id} [get]
func GetChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.Changes.GetChange(ctx, r.PathValue("id"))
	})
}

// @Summary Approve a validated change (self-approval allowed)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/approve [post]
func ApproveChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.approve(ctx, r.PathValue("id"))
	})
}

// @Summary Apply an approved change (transactional)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/apply [post]
func ApplyChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.apply(ctx, r.PathValue("id"))
	})
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

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/reject [post]
func RejectChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.reject(ctx, r.PathValue("id"))
	})
}
