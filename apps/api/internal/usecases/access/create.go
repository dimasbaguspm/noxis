package access

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Create files an access request for the caller.
type Create struct{ usecases.RegistryDeps }

// @Summary Request access to a connection
// @Accept json
// @Param body body entity.AccessRequestCreate true "connection, level and reason"
// @Success 200 {object} entity.AccessRequest
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/access-requests [post]
func (uc Create) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	var body entity.AccessRequestCreate
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if body.Connection == "" || entity.StrongestLevel(body.Level) == "" {
		uc.Fail(ctx, w, fmt.Errorf("connection and a known level are required"))
		return
	}
	if _, err := uc.Connections.Get(ctx, body.Connection); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	open, err := uc.AccessRequests.PendingRequestExists(ctx, principal.Subject, body.Connection)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if open {
		uc.Fail(ctx, w, fmt.Errorf("a request for %s is already pending", body.Connection))
		return
	}
	request := entity.AccessRequest{
		ID: idgen.UUID(), Subject: principal.Subject, Connection: body.Connection, Level: body.Level,
		Reason: strings.TrimSpace(body.Reason), Status: entity.RequestPending, RequestedAt: time.Now().Unix(),
	}
	if err := uc.AccessRequests.CreateAccessRequest(ctx, request); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, request)
}
