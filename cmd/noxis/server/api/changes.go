package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The change routes are shared by SQL and MongoDB, each with its own queue:
// the {engine} path parameter decides which flow runs, and a memory cache
// serves no changes at all.

// @Summary List proposed changes for an engine (newest first)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param connection query string false "substring match on connection name"
// @Param status query string false "exact status match (unverified, validated, approved, applied, failed, rejected)"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{changes=[]entity.Change,total=int}
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/changes [get]
func (s *Server) ListChanges(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		var changes []entity.Change
		switch mod.(type) {
		case interfaces.SQLModule:
			changes, err = s.App.SQLChanges(ctx)
		case interfaces.MongoModule:
			changes, err = s.App.MongoChanges(ctx)
		default:
			return nil, unsupported(mod, "changes")
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
func (s *Server) SubmitChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			var ch entity.Change
			if err := decode(r, &ch); err != nil {
				return nil, err
			}
			return s.App.SubmitChange(ctx, m, ch)
		case interfaces.MongoModule:
			var body mongoChangeBody
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.App.SubmitMongoChange(ctx, m, body.Connection, body.Notes, body.MongoRequest)
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
func (s *Server) GetChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			return s.App.SQLChange(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			return s.App.MongoChange(ctx, r.PathValue("id"))
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
func (s *Server) ApproveChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := s.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return s.App.Approve(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := s.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return s.App.Approve(ctx, r.PathValue("id"))
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
func (s *Server) ApplyChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		switch m := mod.(type) {
		case interfaces.SQLModule:
			if _, err := s.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return s.App.Apply(ctx, m, r.PathValue("id"))
		case interfaces.MongoModule:
			return s.App.ApplyMongo(ctx, m, r.PathValue("id"))
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
func (s *Server) RejectChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := s.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return s.App.Reject(ctx, r.PathValue("id"))
		case interfaces.MongoModule:
			if _, err := s.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
			return s.App.Reject(ctx, r.PathValue("id"))
		}
		return nil, unsupported(mod, "changes")
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
func (s *Server) ScheduleChange(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := s.engine(r)
		if err != nil {
			return nil, err
		}
		var body scheduleBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		switch mod.(type) {
		case interfaces.SQLModule:
			if _, err := s.App.SQLChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
		case interfaces.MongoModule:
			if _, err := s.App.MongoChange(ctx, r.PathValue("id")); err != nil {
				return nil, err
			}
		default:
			return nil, unsupported(mod, "changes")
		}
		return s.App.Schedule(ctx, r.PathValue("id"), body.ScheduledAt)
	})
}

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Param X-API-Key header string true "NOXIS_TICK_KEY"
// @Success 200 {object} application.TickResult
// @Failure 401 {object} map[string]string
// @Router /internal/tick [post]
func (s *Server) PostTick(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		if s.TickKey == "" {
			return nil, fmt.Errorf("%w: scheduling is not configured (set %s)", interfaces.ErrUnsupported, entity.EnvTickKey)
		}
		if r.Header.Get("X-API-Key") != s.TickKey {
			return nil, errUnauthorized
		}
		return s.App.Tick(ctx)
	})
}