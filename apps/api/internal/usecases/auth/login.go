package auth

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Login authenticates a local account.
type Login struct{ usecases.RegistryDeps }

// @Summary Log in with a local account
// @Accept json
// @Param credentials body entity.LoginRequest true "email and password"
// @Success 200 {object} entity.LoginResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/login [post]
func (uc Login) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.LoginRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	token, user, err := uc.Modules.Auth().Login(ctx, body.Email, body.Password)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.LoginResponse{Token: token, User: user})
}
