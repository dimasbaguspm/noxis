// Package change holds the change-queue use cases. SQL and MongoDB keep
// separate queues behind one route: the {engine} path parameter decides which
// flow runs, and a memory cache serves no changes at all.
package change

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
	"noxis/internal/usecases"
)

// unsupportedChangeFlow reports a family that serves no changes queue.
func unsupportedChangeFlow(mod interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, mod.Family())
}

// readable filters a family's changes to the ones the caller may see.
func readable(changes []entity.Change, mod interfaces.Module, allowed map[string]bool) []entity.Change {
	out := make([]entity.Change, 0, len(changes))
	for i := range changes {
		ch := changes[i]
		if !allowed[ch.Connection] {
			continue
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if ch.Kind == constants.KindDML || ch.Kind == constants.KindDDL {
				out = append(out, ch)
			}
		case interfaces.MongoModule:
			if ch.Kind == constants.KindMongo {
				out = append(out, ch)
			}
		}
	}
	return out
}

// filterByConnection keeps changes whose connection name contains the
// case-insensitive needle; an empty needle keeps everything.
func filterByConnection(changes []entity.Change, needle string) []entity.Change {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return changes
	}
	out := make([]entity.Change, 0, len(changes))
	for i := range changes {
		if strings.Contains(strings.ToLower(changes[i].Connection), needle) {
			out = append(out, changes[i])
		}
	}
	return out
}

// filterByStatus keeps changes whose status matches exactly; empty keeps all.
func filterByStatus(changes []entity.Change, status string) []entity.Change {
	status = strings.TrimSpace(status)
	if status == "" {
		return changes
	}
	out := make([]entity.Change, 0, len(changes))
	for i := range changes {
		if changes[i].Status == status {
			out = append(out, changes[i])
		}
	}
	return out
}

// List lists proposed changes for an engine (newest first).
type List struct{ usecases.RegistryDeps }

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{changes=[]entity.Change,total=int}
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	switch mod.(type) {
	case interfaces.SQLModule, interfaces.MongoModule:
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	allowed, err := uc.ReadableConnections(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	changes := readable(all, mod, allowed)
	q := r.URL.Query()
	changes = filterByConnection(changes, q.Get("connection"))
	changes = filterByStatus(changes, q.Get("status"))
	page, perPage := pagination.Params(q.Get("page"), q.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, map[string]any{
		"changes": pagination.Slice(changes, page, perPage),
		"total":   len(changes),
	})
}

// mongoChangeBody is one submitted request: the connection and notes of a
// change, plus the request itself, flat so the console sends one object.
type mongoChangeBody struct {
	Connection string `json:"connection"`
	Notes      string `json:"notes"`
	interfaces.MongoRequest
}

// Submit submits a change (validates before recording).
type Submit struct{ usecases.RegistryDeps }

// @Summary Submit a change (validates before recording)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [post]
func (uc Submit) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		var ch entity.Change
		if err := uc.Decode(r, &ch); err != nil {
			uc.Fail(w, err)
			return
		}
		out, err := uc.submitSQL(ctx, m, ch)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, out)
	case interfaces.MongoModule:
		var body mongoChangeBody
		if err := uc.Decode(r, &body); err != nil {
			uc.Fail(w, err)
			return
		}
		out, err := uc.submitMongo(ctx, m, body.Connection, body.Notes, body.MongoRequest)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, out)
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
	}
}

func (uc Submit) submitSQL(ctx context.Context, sq interfaces.SQLModule, ch entity.Change) (entity.Change, error) {
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := uc.RawConn(ctx, sq, ch.Connection, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.CreatedBy = usecases.Actor(ctx)
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
	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := uc.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

func (uc Submit) submitMongo(ctx context.Context, m interfaces.MongoModule, conn, notes string, req interfaces.MongoRequest) (entity.Change, error) {
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
		CreatedBy:  usecases.Actor(ctx),
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return entity.Change{}, err
	}
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

// Get loads one change for the engine's family.
type Get struct{ usecases.RegistryDeps }

// @Summary Get one change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	var out *entity.Change
	switch mod.(type) {
	case interfaces.SQLModule:
		out, err = uc.SQLChange(r.Context(), r.PathValue("id"))
	case interfaces.MongoModule:
		out, err = uc.MongoChange(r.Context(), r.PathValue("id"))
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// guard loads a change for the engine's family so a SQL URL cannot touch a
// MongoDB request and the reverse.
func (uc Get) guard(ctx context.Context, mod interfaces.Module, id string) error {
	switch mod.(type) {
	case interfaces.SQLModule:
		_, err := uc.SQLChange(ctx, id)
		return err
	case interfaces.MongoModule:
		_, err := uc.MongoChange(ctx, id)
		return err
	}
	return unsupportedChangeFlow(mod)
}

// Approve approves a validated change (self-approval allowed).
type Approve struct{ usecases.RegistryDeps }

// @Summary Approve a validated change (self-approval allowed)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/approve [post]
func (uc Approve) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if err := Get(uc).guard(ctx, mod, r.PathValue("id")); err != nil {
		uc.Fail(w, err)
		return
	}
	ch, err := uc.Changes.GetChange(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(w, err)
		return
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		uc.Fail(w, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status))
		return
	}
	ch.Status, ch.ApprovedAt = constants.ChangeApproved, time.Now().Unix()
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, ch)
}

// Reject rejects a change (kept as an artifact, never deleted).
type Reject struct{ usecases.RegistryDeps }

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/reject [post]
func (uc Reject) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if err := Get(uc).guard(ctx, mod, r.PathValue("id")); err != nil {
		uc.Fail(w, err)
		return
	}
	ch, err := uc.Changes.GetChange(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(w, err)
		return
	}
	if ch.Status == constants.ChangeApplied {
		uc.Fail(w, fmt.Errorf("change already applied — it cannot be rejected"))
		return
	}
	ch.Status, ch.RejectedAt = constants.ChangeRejected, time.Now().Unix()
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, ch)
}

// scheduleBody is the schedule payload: unix seconds, 0 clears the schedule.
type scheduleBody struct {
	ScheduledAt int64 `json:"scheduledAt"`
}

// Schedule schedules or unschedules an approved change.
type Schedule struct{ usecases.RegistryDeps }

// @Summary Schedule or unschedule an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body scheduleBody true "scheduledAt unix seconds; 0 clears the schedule"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/schedule [post]
func (uc Schedule) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	var body scheduleBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	if err := Get(uc).guard(ctx, mod, r.PathValue("id")); err != nil {
		uc.Fail(w, err)
		return
	}
	ch, err := uc.Changes.GetChange(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(w, err)
		return
	}
	if ch.Status != constants.ChangeApproved {
		uc.Fail(w, fmt.Errorf("change is %s, not approved — cannot schedule", ch.Status))
		return
	}
	if body.ScheduledAt > 0 && body.ScheduledAt <= time.Now().Unix() {
		uc.Fail(w, fmt.Errorf("scheduled time must be in the future"))
		return
	}
	ch.ScheduledAt = body.ScheduledAt
	if err := uc.Changes.SaveChange(ctx, *ch); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, ch)
}

// SwaggerMongoChangeBody documents the MongoDB change-submit body: the submit
// route takes either a SQL change or a MongoDB request, and swag cannot express
// that on one operation, so this stub emits the concrete schema.
//
// @Param body body mongoChangeBody true "mongodb request"
func SwaggerMongoChangeBody() {}
