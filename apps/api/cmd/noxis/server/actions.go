package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"noxis/internal/entity"
)

// ChangeAction documents the one route every change transition uses.
// @Summary Approve, reject, apply or schedule a change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body entity.TransitionRequest true "action plus scheduledAt when scheduling"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/action [post]
func ChangeAction() {}

// AccessAction documents the one route every access-request decision uses.
// @Summary Approve or reject an access request
// @Param id path string true "request id"
// @Param body body entity.TransitionRequest true "action plus an optional note"
// @Success 200 {object} entity.AccessRequest
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/access-requests/{id}/action [post]
func AccessAction() {}

// changeAction dispatches a validated action to the existing use case, which
// keeps every status, authorization and schedule rule where it already lived.
func changeAction(d app) http.HandlerFunc {
	return actionHandler(map[string]http.Handler{
		"approve":  http.HandlerFunc(d.ChangeApprove.Serve),
		"reject":   http.HandlerFunc(d.ChangeReject.Serve),
		"apply":    http.HandlerFunc(d.ChangeApply.Serve),
		"schedule": http.HandlerFunc(d.ChangeSchedule.Serve),
	})
}

// accessAction dispatches an access-request decision the same way.
func accessAction(d app) http.HandlerFunc {
	return actionHandler(map[string]http.Handler{
		"approve": http.HandlerFunc(d.AccessApprove.Serve),
		"reject":  http.HandlerFunc(d.AccessReject.Serve),
	})
}

// actionHandler reads the action, then replays the body to the chosen handler:
// a transition payload is small and the target still decodes it itself.
func actionHandler(actions map[string]http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeActionError(w, "cannot read the action payload")
			return
		}
		var body entity.TransitionRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			writeActionError(w, "action payload must be JSON")
			return
		}
		next, ok := actions[body.Action]
		if !ok {
			writeActionError(w, "unknown action "+body.Action)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		next.ServeHTTP(w, r)
	}
}

func writeActionError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: message})
}
