package usecases

import (
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The change routes are shared by SQL and MongoDB, each with its own queue:
// the {engine} path parameter decides which flow runs, and a memory cache
// serves no changes at all. Each Serve resolves the engine, guards the change
// against the right family, then delegates to that family's domain core (Run).

// unsupportedChangeFlow reports a family that serves no changes queue.
func unsupportedChangeFlow(mod interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, mod.Family())
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

// ChangeList lists proposed changes for an engine (newest first).
type ChangeList struct {
	RegistryDeps
}

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{changes=[]entity.Change,total=int}
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [get]
func (uc ChangeList) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	var changes []entity.Change
	switch mod.(type) {
	case interfaces.SQLModule:
		changes, err = SQLChanges(uc).Run(r.Context())
	case interfaces.MongoModule:
		changes, err = MongoChanges(uc).Run(r.Context())
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	if err != nil {
		uc.Fail(w, err)
		return
	}
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

// ChangeSubmit submits a change (validates before recording).
type ChangeSubmit struct {
	RegistryDeps
}

// @Summary Submit a change (validates before recording)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [post]
func (uc ChangeSubmit) Serve(w http.ResponseWriter, r *http.Request) {
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
		out, err := SubmitChange(uc).Run(r.Context(), m, ch)
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
		out, err := SubmitMongoChange(uc).Run(r.Context(), m, body.Connection, body.Notes, body.MongoRequest)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, out)
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
	}
}

// ChangeGet loads one change, insisting it belongs to the engine's family.
type ChangeGet struct {
	RegistryDeps
}

// @Summary Get one change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id} [get]
func (uc ChangeGet) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	var out *entity.Change
	switch mod.(type) {
	case interfaces.SQLModule:
		out, err = SQLChange(uc).Run(r.Context(), r.PathValue("id"))
	case interfaces.MongoModule:
		out, err = MongoChange(uc).Run(r.Context(), r.PathValue("id"))
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

// ChangeApprove approves a validated change (self-approval allowed).
type ChangeApprove struct {
	RegistryDeps
}

// @Summary Approve a validated change (self-approval allowed)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/approve [post]
func (uc ChangeApprove) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	switch mod.(type) {
	case interfaces.SQLModule:
		_, err = SQLChange(uc).Run(r.Context(), r.PathValue("id"))
	case interfaces.MongoModule:
		_, err = MongoChange(uc).Run(r.Context(), r.PathValue("id"))
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	if err != nil {
		uc.Fail(w, err)
		return
	}
	out, err := Approve(uc).Run(r.Context(), r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// ChangeApply applies an approved change.
type ChangeApply struct {
	RegistryDeps
}

// @Summary Apply an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/apply [post]
func (uc ChangeApply) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	id := r.PathValue("id")
	switch m := mod.(type) {
	case interfaces.SQLModule:
		if _, err := (SQLChange(uc)).Run(r.Context(), id); err != nil {
			uc.Fail(w, err)
			return
		}
		out, err := Apply(uc).Run(r.Context(), m, id)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, out)
	case interfaces.MongoModule:
		out, err := ApplyMongo(uc).Run(r.Context(), m, id)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		uc.Respond(w, out)
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
	}
}

// ChangeReject rejects a change (kept as an artifact, never deleted).
type ChangeReject struct {
	RegistryDeps
}

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/reject [post]
func (uc ChangeReject) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	switch mod.(type) {
	case interfaces.SQLModule:
		_, err = SQLChange(uc).Run(r.Context(), r.PathValue("id"))
	case interfaces.MongoModule:
		_, err = MongoChange(uc).Run(r.Context(), r.PathValue("id"))
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	if err != nil {
		uc.Fail(w, err)
		return
	}
	out, err := Reject(uc).Run(r.Context(), r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// scheduleBody is the schedule payload: unix seconds, 0 clears the schedule.
type scheduleBody struct {
	ScheduledAt int64 `json:"scheduledAt"`
}

// ChangeSchedule schedules or unschedules an approved change.
type ChangeSchedule struct {
	RegistryDeps
}

// @Summary Schedule or unschedule an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body scheduleBody true "scheduledAt unix seconds; 0 clears the schedule"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/schedule [post]
func (uc ChangeSchedule) Serve(w http.ResponseWriter, r *http.Request) {
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
	switch mod.(type) {
	case interfaces.SQLModule:
		if _, err := (SQLChange(uc)).Run(r.Context(), r.PathValue("id")); err != nil {
			uc.Fail(w, err)
			return
		}
	case interfaces.MongoModule:
		if _, err := (MongoChange(uc)).Run(r.Context(), r.PathValue("id")); err != nil {
			uc.Fail(w, err)
			return
		}
	default:
		uc.Fail(w, unsupportedChangeFlow(mod))
		return
	}
	out, err := Schedule(uc).Run(r.Context(), r.PathValue("id"), body.ScheduledAt)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// Tick applies every due scheduled change; the external scheduler drives it.
// @Summary Apply every due scheduled change (external cron / scheduler)
// @Success 200 {object} TickResult
// @Router /internal/callback/scheduler [post]
func (uc Tick) Serve(w http.ResponseWriter, r *http.Request) {
	out, err := uc.Run(r.Context())
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}
