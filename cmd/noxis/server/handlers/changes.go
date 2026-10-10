package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// The change routes are shared by SQL and MongoDB, each with its own queue:
// the {engine} path parameter decides which flow runs, and a memory cache
// serves no changes at all.

// unsupportedChangeFlow reports a family that serves no changes queue.
func unsupportedChangeFlow(mod interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", interfaces.ErrUnsupported, mod.Family())
}

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{changes=[]entity.Change,total=int}
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes [get]
func ListChanges(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		var changes []entity.Change
		switch mod.(type) {
		case interfaces.SQLModule:
			changes, err = d.App.SQLChanges(ctx)
		case interfaces.MongoModule:
			changes, err = d.App.MongoChanges(ctx)
		default:
			return nil, unsupportedChangeFlow(mod)
		}
		if err != nil {
			return nil, err
		}
		changes = filterByConnection(changes, r.URL.Query().Get("connection"))
		changes = filterByStatus(changes, r.URL.Query().Get("status"))
		page, perPage := readPagination(r, 20)
		return map[string]any{
			"changes": pageSlice(changes, page, perPage),
			"total":   len(changes),
		}, nil
	})
}

// filterByConnection keeps the changes whose connection name contains the
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

// filterByStatus keeps the changes whose status matches exactly; an empty
// needle keeps everything.
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

// @Summary Submit a change (validates before recording)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes [post]
func SubmitChange(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			var ch entity.Change
			if err := decode(r, &ch); err != nil {
				return nil, err
			}
			return d.App.SubmitChange(ctx, m, ch)
		case interfaces.MongoModule:
			var body mongoChangeBody
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return d.App.SubmitMongoChange(ctx, m, body.Connection, body.Notes, body.MongoRequest)
		}
		return nil, unsupportedChangeFlow(mod)
	})
}

// @Summary Get one change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id} [get]
func GetChange(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			return d.App.SQLChange(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			return d.App.MongoChange(ctx, r.PathValue("id"))
		}
		return nil, unsupportedChangeFlow(mod)
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
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.App.Approve(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := d.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.App.Approve(ctx, r.PathValue("id"))
		}
		return nil, unsupportedChangeFlow(mod)
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
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.App.Apply(ctx, m, r.PathValue("id"))
		case interfaces.MongoModule:
			return d.App.ApplyMongo(ctx, m, r.PathValue("id"))
		}
		return nil, unsupportedChangeFlow(mod)
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
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.App.Reject(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := d.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return d.App.Reject(ctx, r.PathValue("id"))
		}
		return nil, unsupportedChangeFlow(mod)
	})
}

type scheduleBody struct {
	ScheduledAt int64 `json:"scheduledAt"`
}

// @Summary Schedule or unschedule an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body scheduleBody true "scheduledAt unix seconds; 0 clears the schedule"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes/{id}/schedule [post]
func ScheduleChange(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.App.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		var body scheduleBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := d.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
		case interfaces.MongoModule:
			if _, err := d.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
		default:
			return nil, unsupportedChangeFlow(mod)
		}
		return d.App.Schedule(ctx, r.PathValue("id"), body.ScheduledAt)
	})
}

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Success 200 {object} usecases.TickResult
// @Router /internal/callback/scheduler [post]
func PostScheduler(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		return d.App.Tick(ctx)
	})
}
