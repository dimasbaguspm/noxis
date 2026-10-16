package handlers

import (
	"context"
	"errors"
	"net/http"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

var errLocalAccountRequired = errors.New("a local account is required for this action")

// @Summary Public auth configuration (console bootstrap)
// @Success 200 {object} config.AuthInfo
// @Router /external/v1/auth/config [get]
func GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(context.Context) (any, error) {
		return d.Auth, nil
	})
}

// loginRequest is the local credentials payload.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the token the console stores and the account it belongs to.
type loginResponse struct {
	Token string      `json:"token"`
	User  entity.User `json:"user"`
}

// @Summary Log in with a local account
// @Accept json
// @Param credentials body loginRequest true "username and password"
// @Success 200 {object} loginResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/login [post]
func PostLogin(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		var body loginRequest
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		token, user, err := d.Modules.Auth().Login(ctx, body.Username, body.Password)
		if err != nil {
			return nil, err
		}
		return loginResponse{Token: token, User: user}, nil
	})
}

// @Summary Log out the current local account
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/logout [post]
func PostLogout(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handleNoContent(w, r, func(ctx context.Context) error {
		p, ok := usecases.Principal(ctx)
		if !ok {
			return errUnauthorized
		}
		return d.Modules.Auth().Logout(ctx, p.Subject)
	})
}

// changePasswordRequest is the self-serve password payload.
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// @Summary Change the current local account's password
// @Accept json
// @Param passwords body changePasswordRequest true "old and new password"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/password [post]
func PostChangePassword(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handleNoContent(w, r, func(ctx context.Context) error {
		p, ok := usecases.Principal(ctx)
		if !ok {
			return errUnauthorized
		}
		if entity.LocalUsername(p.Subject) == "" {
			return errLocalAccountRequired
		}
		var body changePasswordRequest
		if err := decode(r, &body); err != nil {
			return err
		}
		return d.Modules.Auth().ChangePassword(ctx, p.Subject, body.OldPassword, body.NewPassword)
	})
}

// MeResponse is the caller plus the effective level it holds on each readable
// connection, so the console can gate actions without guessing.
type MeResponse struct {
	entity.Identity
	Connections map[string]string `json:"connections"`
}

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		p, ok := usecases.Principal(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		levels, err := d.ConnectionLevels.Serve(ctx)
		if err != nil {
			return nil, err
		}
		return MeResponse{Identity: p, Connections: levels}, nil
	})
}
