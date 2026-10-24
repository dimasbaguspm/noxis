package server

import (
	"net/http"

	"noxis/internal/usecases"
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
	return usecases.ActionHandler(map[string]http.Handler{
		"approve":  http.HandlerFunc(d.ChangeApprove.Serve),
		"reject":   http.HandlerFunc(d.ChangeReject.Serve),
		"apply":    http.HandlerFunc(d.ChangeApply.Serve),
		"schedule": http.HandlerFunc(d.ChangeSchedule.Serve),
	})
}

// accessAction dispatches an access-request decision the same way.
func accessAction(d app) http.HandlerFunc {
	return usecases.ActionHandler(map[string]http.Handler{
		"approve": http.HandlerFunc(d.AccessApprove.Serve),
		"reject":  http.HandlerFunc(d.AccessReject.Serve),
	})
}
