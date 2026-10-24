package auth

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Logout ends the caller's session.
type Logout struct{ usecases.RegistryDeps }

// @Summary Log out the current local account
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/logout [post]
func (uc Logout) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	if err := uc.Modules.Auth().Logout(ctx, principal.Subject); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}
