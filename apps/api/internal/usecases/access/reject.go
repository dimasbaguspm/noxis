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

// Reject closes an access request without granting anything.
type Reject struct{ usecases.RegistryDeps }

func (uc Reject) Serve(w http.ResponseWriter, r *http.Request) {
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
	now := time.Now().Unix()
	if err := uc.Repositories.AccessRequests.SetAccessRequestDecision(ctx, request.ID, entity.RequestRejected, principal.Subject, now, body.Note); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	request.Status, request.DecidedBy, request.DecidedAt, request.DecisionNote = entity.RequestRejected, principal.Subject, now, body.Note
	uc.Respond(w, request)
}
