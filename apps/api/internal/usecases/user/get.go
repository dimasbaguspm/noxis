package user

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Get returns one account, its access, and its activity.
type Get struct{ usecases.RegistryDeps }

// @Summary One account with its access and activity (admin only)
// @Param subject path string true "user subject"
// @Success 200 {object} entity.UserDetail
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	user, err := uc.Users.GetUser(ctx, r.PathValue("subject"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	access, err := uc.Permissions.ListPermissionsForMembers(ctx, append([]string{user.Subject}, user.Groups...))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	events, err := uc.AccessEvents.ListAccessEvents(ctx, user.Subject, 50)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if access == nil {
		access = []entity.Permission{}
	}
	if events == nil {
		events = []entity.AccessEvent{}
	}
	uc.Respond(w, entity.UserDetail{User: *user, Access: access, Events: events})
}
