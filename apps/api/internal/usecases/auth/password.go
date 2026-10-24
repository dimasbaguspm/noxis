package auth

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Password lets a local account change its own password.
type Password struct{ usecases.RegistryDeps }

// @Summary Change the current local account's password
// @Accept json
// @Param passwords body entity.PasswordChange true "old and new password"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/password [post]
func (uc Password) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	if entity.LocalUsername(principal.Subject) == "" {
		uc.Fail(ctx, w, constants.ErrForbidden)
		return
	}
	var body entity.PasswordChange
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Auth.ChangePassword(ctx, principal.Subject, body.OldPassword, body.NewPassword); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}
