// Package change holds the change-queue use cases. SQL and MongoDB keep
// separate queues behind one route: the {engine} path parameter decides which
// flow runs, and a memory cache serves no changes at all. The rules live in
// modules/changes; these use cases authorize, translate and answer.
package change

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/common/logging"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// supports reports whether a family serves a change queue at all.
func supports(mod interfaces.Module) bool {
	switch mod.(type) {
	case interfaces.SQLModule, interfaces.MongoModule:
		return true
	}
	return false
}

// readable filters a family's changes to the ones the caller may see.
func readable(all []entity.Change, mod interfaces.Module, allowed map[string]bool) []entity.Change {
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		ch := all[i]
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
func filterByConnection(all []entity.Change, needle string) []entity.Change {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if strings.Contains(strings.ToLower(all[i].Connection), needle) {
			out = append(out, all[i])
		}
	}
	return out
}

// filterByStatus keeps changes whose status matches exactly; empty keeps all.
func filterByStatus(all []entity.Change, status string) []entity.Change {
	status = strings.TrimSpace(status)
	if status == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Status == status {
			out = append(out, all[i])
		}
	}
	return out
}

// List lists proposed changes for an engine (newest first).
type List struct{ usecases.RegistryDeps }

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applying, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} entity.ChangeList
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !supports(mod) {
		uc.Fail(ctx, w, unsupportedChangeFlow(mod))
		return
	}
	all, err := uc.Changes.ListChanges(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	allowed, err := uc.ReadableConnections(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	q := r.URL.Query()
	visible := readable(all, mod, allowed)
	visible = filterByConnection(visible, q.Get("connection"))
	visible = filterByStatus(visible, q.Get("status"))
	page, perPage := pagination.Params(q.Get("page"), q.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, entity.ChangeList{Changes: pagination.Slice(visible, page, perPage), Total: len(visible)})
}

// unsupportedChangeFlow reports a family that serves no changes queue.
func unsupportedChangeFlow(mod interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, mod.Family())
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
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		var ch entity.Change
		if err := uc.Decode(r, &ch); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		conn, err := uc.RawConn(ctx, m, ch.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := uc.Modules.Changes().SQL.Submit(ctx, *conn, entity.Actor(ctx), ch)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, mod.Family(), r.PathValue("engine"))
		uc.Respond(w, out)
	case interfaces.MongoModule:
		var body entity.MongoChangeRequest
		if err := uc.Decode(r, &body); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		conn, err := uc.RawConn(ctx, m, body.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := uc.Modules.Changes().Mongo.Submit(ctx, *conn, entity.Actor(ctx), body.Notes, body.MongoRequest)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, mod.Family(), r.PathValue("engine"))
		uc.Respond(w, out)
	default:
		uc.Fail(ctx, w, unsupportedChangeFlow(mod))
	}
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
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	ch, err := changeFor(ctx, uc.RegistryDeps, mod, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, ch)
}

// changeFor loads one change for the addressed family, after the family gate and
// the caller's read grant. Every mutating flow starts here, so a SQL URL cannot
// touch a MongoDB request and the reverse.
func changeFor(ctx context.Context, uc usecases.RegistryDeps, mod interfaces.Module, id string) (*entity.Change, error) {
	switch mod.(type) {
	case interfaces.SQLModule:
		return uc.SQLChange(ctx, id)
	case interfaces.MongoModule:
		return uc.MongoChange(ctx, id)
	}
	return nil, unsupportedChangeFlow(mod)
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
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	decide(ctx, uc.RegistryDeps, w, r, func(ch *entity.Change) error { return uc.Modules.Changes().SQL.Approve(ctx, ch, entity.Actor(ctx)) })
}

// Reject rejects a change (kept as an artifact, never deleted).
type Reject struct{ usecases.RegistryDeps }

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/reject [post]
func (uc Reject) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	decide(ctx, uc.RegistryDeps, w, r, func(ch *entity.Change) error { return uc.Modules.Changes().SQL.Reject(ctx, ch, entity.Actor(ctx)) })
}

// decide is the shared shape of a decision route: resolve the family, load the
// change, require admin on its connection, then let the lifecycle apply the rule.
func decide(ctx context.Context, uc usecases.RegistryDeps, w http.ResponseWriter, r *http.Request, apply func(*entity.Change) error) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	ch, err := changeFor(ctx, uc, mod, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := apply(ch); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, ch)
}

// Schedule schedules or unschedules an approved change.
type Schedule struct{ usecases.RegistryDeps }

// @Summary Schedule or unschedule an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body entity.ScheduleRequest true "scheduledAt unix seconds; 0 clears the schedule"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/schedule [post]
func (uc Schedule) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.ScheduleRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	decide(ctx, uc.RegistryDeps, w, r, func(ch *entity.Change) error {
		if ch.Kind == constants.KindMongo {
			return uc.Modules.Changes().Mongo.Schedule(ctx, ch, body.ScheduledAt, entity.Actor(ctx))
		}
		return uc.Modules.Changes().SQL.Schedule(ctx, ch, body.ScheduledAt, entity.Actor(ctx))
	})
}

// SwaggerMongoChangeBody documents the MongoDB change-submit body: the submit
// route takes either a SQL change or a MongoDB request, and swag cannot express
// that on one operation, so this stub emits the concrete schema.
//
// @Param body body entity.MongoChangeRequest true "mongodb request"
func SwaggerMongoChangeBody() {}
