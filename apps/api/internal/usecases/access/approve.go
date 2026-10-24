package access

import (
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Approve grants the requested level for the standard 120 days.
type Approve struct{ usecases.RegistryDeps }

// @Summary Approve an access request
// @Param id path string true "request id"
// @Param body body entity.AccessDecision false "optional note"
// @Success 200 {object} entity.AccessRequest
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/access-requests/{id}/approve [post]
func (uc Approve) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	request, err := uc.Repositories.AccessRequests.GetAccessRequest(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if request.Status != entity.RequestPending {
		uc.Fail(ctx, w, fmt.Errorf("request is %s, not pending", request.Status))
		return
	}
	if err := canDecide(ctx, uc.RegistryDeps, request, principal); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.AccessDecision
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	now := time.Now()
	grant := entity.Permission{
		Connection: request.Connection, Member: request.Subject, MemberType: entity.MemberUser, Level: request.Level,
		ExpiresAt: now.Add(uc.GrantTTL()).Unix(), CreatedBy: principal.Subject,
		CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}
	if err := uc.Repositories.Permissions.SetPermission(ctx, grant); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Repositories.AccessRequests.SetAccessRequestDecision(ctx, request.ID, entity.RequestApproved, principal.Subject, now.Unix(), body.Note); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	request.Status, request.DecidedBy, request.DecidedAt, request.DecisionNote = entity.RequestApproved, principal.Subject, now.Unix(), body.Note
	request.ExpiresAt = grant.ExpiresAt
	uc.Respond(w, request)
}
