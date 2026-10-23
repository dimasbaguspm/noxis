// Package access holds access requests: a user asks for a level on one
// connection, and an admin or that instance's admin decides.
package access

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// createBody is the request payload.
type createBody struct {
	Connection string `json:"connection"`
	Level      string `json:"level"`
	Reason     string `json:"reason"`
}

// Create files an access request for the caller.
type Create struct{ usecases.RegistryDeps }

// @Summary Request access to a connection
// @Accept json
// @Param body body createBody true "connection, level and reason"
// @Success 200 {object} entity.AccessRequest
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/access-requests [post]
func (uc Create) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := usecases.Principal(ctx)
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	var body createBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	if body.Connection == "" || entity.StrongestLevel(body.Level) == "" {
		uc.Fail(w, fmt.Errorf("connection and a known level are required"))
		return
	}
	if _, err := uc.Connections.Get(ctx, body.Connection); err != nil {
		uc.Fail(w, err)
		return
	}
	open, err := uc.AccessRequests.PendingRequestExists(ctx, p.Subject, body.Connection)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if open {
		uc.Fail(w, fmt.Errorf("a request for %s is already pending", body.Connection))
		return
	}
	req := entity.AccessRequest{
		ID: idgen.UUID(), Subject: p.Subject, Connection: body.Connection, Level: body.Level,
		Reason: strings.TrimSpace(body.Reason), Status: entity.RequestPending, RequestedAt: time.Now().Unix(),
	}
	if err := uc.AccessRequests.CreateAccessRequest(ctx, req); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, req)
}

// List returns the caller's requests (scope=mine, the default) or the pending
// requests the caller may decide (scope=to-approve).
type List struct{ usecases.RegistryDeps }

// @Summary List access requests (mine, or the ones I may decide)
// @Param scope query string false "mine (default) or to-approve"
// @Success 200 {object} object{requests=[]entity.AccessRequest}
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/access-requests [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := usecases.Principal(ctx)
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	if r.URL.Query().Get("scope") != "to-approve" {
		list, err := uc.AccessRequests.ListAccessRequests(ctx, p.Subject)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		if list == nil {
			list = []entity.AccessRequest{}
		}
		uc.Respond(w, map[string]any{"requests": list})
		return
	}
	var connections []string
	if !p.IsAdmin() {
		all, err := uc.Connections.List(ctx)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		connections = []string{}
		for i := range all {
			level, err := uc.EffectiveLevel(ctx, all[i].Name, p)
			if err != nil {
				uc.Fail(w, err)
				return
			}
			if entity.LevelAllows(level, entity.LevelAdmin) {
				connections = append(connections, all[i].Name)
			}
		}
	}
	list, err := uc.AccessRequests.ListPendingRequests(ctx, connections)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if list == nil {
		list = []entity.AccessRequest{}
	}
	uc.Respond(w, map[string]any{"requests": list})
}

// decisionBody is the approve/reject payload.
type decisionBody struct {
	Note string `json:"note"`
}

// canDecide enforces who may decide a request: never the requester, then a
// global admin or a caller holding admin on that connection.
func canDecide(ctx context.Context, d usecases.RegistryDeps, req *entity.AccessRequest, p entity.Identity) error {
	if p.Subject == req.Subject {
		return fmt.Errorf("%w: you cannot decide your own request", constants.ErrForbidden)
	}
	if p.IsAdmin() {
		return nil
	}
	level, err := d.EffectiveLevel(ctx, req.Connection, p)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, entity.LevelAdmin) {
		return fmt.Errorf("%w: deciding %s requires admin on %q", constants.ErrForbidden, req.ID, req.Connection)
	}
	return nil
}

// Approve grants the requested level for the standard 120 days and closes the
// request with the decision recorded.
type Approve struct{ usecases.RegistryDeps }

// @Summary Approve an access request
// @Param id path string true "request id"
// @Param body body decisionBody false "optional note"
// @Success 200 {object} entity.AccessRequest
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/access-requests/{id}/approve [post]
func (uc Approve) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := usecases.Principal(ctx)
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	req, err := uc.AccessRequests.GetAccessRequest(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if req.Status != entity.RequestPending {
		uc.Fail(w, fmt.Errorf("request is %s, not pending", req.Status))
		return
	}
	if err := canDecide(ctx, uc.RegistryDeps, req, p); err != nil {
		uc.Fail(w, err)
		return
	}
	var body decisionBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	now := time.Now()
	grant := entity.Permission{
		Connection: req.Connection, Member: req.Subject, MemberType: entity.MemberUser, Level: req.Level,
		ExpiresAt: now.Add(uc.GrantTTL()).Unix(), CreatedBy: p.Subject,
		CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}
	if err := uc.Permissions.SetPermission(ctx, grant); err != nil {
		uc.Fail(w, err)
		return
	}
	if err := uc.AccessRequests.SetAccessRequestDecision(ctx, req.ID, entity.RequestApproved, p.Subject, now.Unix(), body.Note); err != nil {
		uc.Fail(w, err)
		return
	}
	req.Status, req.DecidedBy, req.DecidedAt, req.DecisionNote = entity.RequestApproved, p.Subject, now.Unix(), body.Note
	req.ExpiresAt = grant.ExpiresAt
	uc.Respond(w, req)
}

// Reject closes an access request without granting anything.
type Reject struct{ usecases.RegistryDeps }

// @Summary Reject an access request
// @Param id path string true "request id"
// @Param body body decisionBody false "optional note"
// @Success 200 {object} entity.AccessRequest
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/access-requests/{id}/reject [post]
func (uc Reject) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := usecases.Principal(ctx)
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	req, err := uc.AccessRequests.GetAccessRequest(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if req.Status != entity.RequestPending {
		uc.Fail(w, fmt.Errorf("request is %s, not pending", req.Status))
		return
	}
	if err := canDecide(ctx, uc.RegistryDeps, req, p); err != nil {
		uc.Fail(w, err)
		return
	}
	var body decisionBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	now := time.Now().Unix()
	if err := uc.AccessRequests.SetAccessRequestDecision(ctx, req.ID, entity.RequestRejected, p.Subject, now, body.Note); err != nil {
		uc.Fail(w, err)
		return
	}
	req.Status, req.DecidedBy, req.DecidedAt, req.DecisionNote = entity.RequestRejected, p.Subject, now, body.Note
	uc.Respond(w, req)
}
