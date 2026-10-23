// Package auth holds the authentication and identity use cases.
package auth

import (
	"net/http"

	"noxis/internal/common/config"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Config is the public auth configuration the console bootstraps from.
type Config struct {
	usecases.RegistryDeps
	Auth config.AuthInfo
}

// @Summary Public auth configuration (console bootstrap)
// @Success 200 {object} config.AuthInfo
// @Router /external/v1/auth/config [get]
func (uc Config) Serve(w http.ResponseWriter, _ *http.Request) { uc.Respond(w, uc.Auth) }

// loginRequest is the local credentials payload. Email is the account
// identifier: the env root signs in with the address in NOXIS_ADMIN_EMAIL.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the token the console stores and the account it belongs to.
type loginResponse struct {
	Token string      `json:"token"`
	User  entity.User `json:"user"`
}

// Login authenticates a local account.
type Login struct{ usecases.RegistryDeps }

// @Summary Log in with a local account
// @Accept json
// @Param credentials body loginRequest true "email and password"
// @Success 200 {object} loginResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/login [post]
func (uc Login) Serve(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	token, user, err := uc.Modules.Auth().Login(r.Context(), body.Email, body.Password)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, loginResponse{Token: token, User: user})
}

// Logout ends the caller's session.
type Logout struct{ usecases.RegistryDeps }

// @Summary Log out the current local account
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/logout [post]
func (uc Logout) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := usecases.Principal(r.Context())
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	if err := uc.Modules.Auth().Logout(r.Context(), p.Subject); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.RespondNoContent(w)
}

// changePasswordRequest is the self-serve password payload.
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// Password lets a local account change its own password.
type Password struct{ usecases.RegistryDeps }

// @Summary Change the current local account's password
// @Accept json
// @Param passwords body changePasswordRequest true "old and new password"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/password [post]
func (uc Password) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := usecases.Principal(r.Context())
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	if entity.LocalUsername(p.Subject) == "" {
		uc.Fail(w, constants.ErrForbidden)
		return
	}
	var body changePasswordRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	if err := uc.Modules.Auth().ChangePassword(r.Context(), p.Subject, body.OldPassword, body.NewPassword); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.RespondNoContent(w)
}

// AccessGrant is the caller's effective level on a connection and when it
// lapses. A super admin's grant never expires (ExpiresAt 0).
type AccessGrant struct {
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expiresAt"`
}

// MeResponse is the caller plus the effective grant it holds on each readable
// connection, so the console can gate actions without guessing.
type MeResponse struct {
	entity.Identity
	Connections map[string]AccessGrant `json:"connections"`
}

// Me returns the authenticated caller and its per-connection levels.
type Me struct{ usecases.RegistryDeps }

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/me [get]
func (uc Me) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, ok := usecases.Principal(ctx)
	if !ok {
		uc.Fail(w, constants.ErrUnauthorized)
		return
	}
	grants := map[string]AccessGrant{}
	list, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	for i := range list {
		c := &list[i]
		if p.Super {
			grants[c.Name] = AccessGrant{Level: entity.LevelAdmin}
			continue
		}
		level, expiresAt, err := uc.EffectiveGrant(ctx, c.Name, p)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		if level != "" {
			grants[c.Name] = AccessGrant{Level: level, ExpiresAt: expiresAt}
		}
	}
	uc.Respond(w, MeResponse{Identity: p, Connections: grants})
}
