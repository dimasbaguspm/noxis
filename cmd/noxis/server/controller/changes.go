package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/services/sql"
)

// decode reads a JSON body into v.
func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// atoi is a lenient query-param int (0 on absence/garbage; the SQL module
// applies its own default/clamp).
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// changes lists every proposed change, newest first.
func (d Deps) changes(ctx context.Context) ([]entity.Change, error) {
	return d.Changes.ListChanges(ctx)
}

// change returns one change.
func (d Deps) change(ctx context.Context, id string) (entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// submitChange records a change and validates it: the query must be a write,
// and the query plus its rollback are dry-run together inside a transaction
// that always rolls back — proving the rollback undoes the change. Nothing
// persists on submit.
func (d Deps) submitChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := d.raw(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.Status = constants.ChangeValidated
	ch.Checks = nil

	kind := entity.ChangeCheck{Target: "query kind"}
	if err := d.SQL.VerifyWrite(ch.Query); err != nil {
		kind.Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK = true
		kind.Detail = "write statements only"
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := d.SQL.ValidateChange(ctx, *conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK = true
		dry.Detail = res.Message
	case isUnverifiable(err):
		ch.Status = constants.ChangeUnverified
		dry.Error = err.Error()
	default:
		ch.Status = constants.ChangeFailed
		dry.Error = err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	if err := d.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// approve signs off a validated (or unverifiable) change; self-approval is
// allowed — this is a single-operator tool.
func (d Deps) approve(ctx context.Context, id string) (entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return *ch, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status = constants.ChangeApproved
	ch.ApprovedAt = time.Now().Unix()
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// apply executes an approved change transactionally and records the outcome
// (a failure stays on the change, not just in the HTTP response).
func (d Deps) apply(ctx context.Context, id string) (entity.Change, error) {
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
	ch.Status = constants.ChangeApplied
	ch.AppliedAt = time.Now().Unix()
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// isUnverifiable reports the driver-cannot-dry-run case without importing
// the SQL module's errors into the handler flow.
func isUnverifiable(err error) bool { return sql.IsUnverifiable(err) }

// ---- handlers ----

// @Summary List proposed changes (newest first)
// @Success 200 {object} object{changes=[]entity.Change}
// @Router /v1/changes [get]
func ListChanges(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.list")
	defer end(nil)
	changes, err := d.changes(ctx)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changes": changes})
}

// @Summary Submit a change (validates query + rollback; nothing persists)
// @Param body body entity.Change true "connection, notes, query, rollback"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes [post]
func SubmitChange(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.submit")
	defer end(nil)
	var ch entity.Change
	if err := decode(r, &ch); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	saved, err := d.submitChange(ctx, ch)
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// @Summary Get one change
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/changes/{id} [get]
func GetChange(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.get")
	defer end(nil)
	ch, err := d.change(ctx, r.PathValue("id"))
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Approve a validated change (self-approval allowed)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/approve [post]
func ApproveChange(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.approve")
	defer end(nil)
	ch, err := d.approve(ctx, r.PathValue("id"))
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Apply an approved change (transactional)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id}/apply [post]
func ApplyChange(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.apply")
	defer end(nil)
	ch, err := d.apply(ctx, r.PathValue("id"))
	if err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// @Summary Reject (discard) a change
// @Param id path string true "change id"
// @Success 204 "discarded"
// @Failure 400 {object} map[string]string
// @Router /v1/changes/{id} [delete]
func RejectChange(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context(), "changes.reject")
	defer end(nil)
	if err := d.Changes.DeleteChange(ctx, r.PathValue("id")); err != nil {
		end(err)
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
