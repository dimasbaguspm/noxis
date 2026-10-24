package user

import (
	"fmt"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Update changes an account's status and, for the super admin, global role.
type Update struct{ usecases.RegistryDeps }

// @Summary Ban/enable an account, or promote/demote an admin (admin only; role changes are super only)
// @Param subject path string true "user subject"
// @Param body body entity.UserUpdate true "status and/or role"
// @Success 200 {object} entity.User
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [patch]
func (uc Update) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	caller, _ := entity.Principal(ctx)
	subject := r.PathValue("subject")
	var body entity.UserUpdate
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if body.Status == nil && body.Role == nil {
		uc.Fail(ctx, w, fmt.Errorf("status or role is required"))
		return
	}
	user, err := uc.Repositories.Users.GetUser(ctx, subject)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.checkPatch(caller, user, body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if subject == caller.Subject {
		uc.Fail(ctx, w, fmt.Errorf("%w: you cannot change your own account", constants.ErrForbidden))
		return
	}
	if user.IsSuper() {
		uc.Fail(ctx, w, fmt.Errorf("%w: the super admin cannot be changed", constants.ErrForbidden))
		return
	}
	if err := uc.applyRole(ctx, caller, user, body.Role); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.applyStatus(ctx, caller, user, body.Status); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, user)
}
