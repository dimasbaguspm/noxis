// Package auth holds the authentication and identity use cases.
package auth

import (
	"net/http"

	"noxis/internal/constants"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Config is the public auth configuration the console bootstraps from.
type Config struct {
	usecases.RegistryDeps
	Auth entity.AuthConfig
}

// @Summary Public auth configuration (console bootstrap)
// @Success 200 {object} entity.AuthConfig
// @Router /external/v1/auth/config [get]
func (uc Config) Serve(w http.ResponseWriter, r *http.Request) {
	_, end := logging.StartSpan(r.Context())
	defer end(nil)
	uc.Respond(w, uc.Auth)
}

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

// Logout ends the caller's session.
type Logout struct{ usecases.RegistryDeps }

// @Summary Log out the current local account
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/logout [post]
func (uc Logout) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	p, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	if err := uc.Modules.Auth().Logout(ctx, p.Subject); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}

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
	p, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	if entity.LocalUsername(p.Subject) == "" {
		uc.Fail(ctx, w, constants.ErrForbidden)
		return
	}
	var body entity.PasswordChange
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Modules.Auth().ChangePassword(ctx, p.Subject, body.OldPassword, body.NewPassword); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}

// Me returns the authenticated caller and its per-connection levels.
type Me struct{ usecases.RegistryDeps }

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} entity.MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/me [get]
func (uc Me) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	p, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	grants := map[string]entity.AccessGrant{}
	list, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	for i := range list {
		c := &list[i]
		if p.IsSuper() {
			grants[c.Name] = entity.AccessGrant{Level: entity.LevelAdmin}
			continue
		}
		level, expiresAt, err := uc.EffectiveGrant(ctx, c.Name, p)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if level != "" {
			grants[c.Name] = entity.AccessGrant{Level: level, ExpiresAt: expiresAt}
		}
	}
	uc.Respond(w, entity.MeResponse{Identity: p, Connections: grants})
}
