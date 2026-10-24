package access

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List returns the caller's requests or the pending requests they may decide.
type List struct{ usecases.RegistryDeps }

// @Summary List access requests (mine, or the ones I may decide)
// @Param scope query string false "mine (default) or to-approve"
// @Success 200 {object} entity.AccessRequestList
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/access-requests [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	if r.URL.Query().Get("scope") != "to-approve" {
		requests, err := uc.AccessRequests.ListAccessRequests(ctx, principal.Subject)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if requests == nil {
			requests = []entity.AccessRequest{}
		}
		uc.Respond(w, entity.AccessRequestList{Requests: requests})
		return
	}
	var connections []string
	if !principal.IsAdmin() {
		all, err := uc.Connections.List(ctx)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connections = []string{}
		for i := range all {
			level, err := uc.EffectiveLevel(ctx, all[i].Name, principal)
			if err != nil {
				uc.Fail(ctx, w, err)
				return
			}
			if entity.LevelAllows(level, entity.LevelAdmin) {
				connections = append(connections, all[i].Name)
			}
		}
	}
	requests, err := uc.AccessRequests.ListPendingRequests(ctx, connections)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if requests == nil {
		requests = []entity.AccessRequest{}
	}
	uc.Respond(w, entity.AccessRequestList{Requests: requests})
}
