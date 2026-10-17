// Package user holds the super-only user directory use cases.
package user

import (
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List is the directory: every account, filtered by a search term and status.
type List struct{ usecases.RegistryDeps }

// @Summary List accounts (super admin only)
// @Param q query string false "match subject or username"
// @Param status query string false "all, active or disabled"
// @Success 200 {object} object{users=[]entity.User}
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/users [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(w, err)
		return
	}
	users, err := uc.Users.ListUsers(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := r.URL.Query().Get("status")
	out := make([]entity.User, 0, len(users))
	for i := range users {
		u := &users[i]
		if q != "" && !strings.Contains(strings.ToLower(u.Subject), q) && !strings.Contains(strings.ToLower(u.Username), q) {
			continue
		}
		if status == "active" && u.Disabled {
			continue
		}
		if status == "disabled" && !u.Disabled {
			continue
		}
		out = append(out, *u)
	}
	uc.Respond(w, map[string]any{"users": out})
}

// Detail is one account with the access it holds and its activity trail.
type Detail struct {
	entity.User
	Access []entity.Permission  `json:"access"`
	Events []entity.AccessEvent `json:"events"`
}

// Get returns one account, its access (direct and via groups) and its activity.
type Get struct{ usecases.RegistryDeps }

// @Summary One account with its access and activity (super admin only)
// @Param subject path string true "user subject"
// @Success 200 {object} Detail
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(w, err)
		return
	}
	u, err := uc.Users.GetUser(ctx, r.PathValue("subject"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	access, err := uc.Permissions.ListPermissionsForMembers(ctx, append([]string{u.Subject}, u.Groups...))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	events, err := uc.AccessEvents.ListAccessEvents(ctx, u.Subject, 50)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if access == nil {
		access = []entity.Permission{}
	}
	if events == nil {
		events = []entity.AccessEvent{}
	}
	uc.Respond(w, Detail{User: *u, Access: access, Events: events})
}

// updateBody is the enable/disable payload.
type updateBody struct {
	Disabled *bool `json:"disabled"`
}

// Update disables or re-enables an account. The super admin and the caller are
// never disableable, so a store cannot lock itself out.
type Update struct{ usecases.RegistryDeps }

// @Summary Disable or re-enable an account (super admin only)
// @Param subject path string true "user subject"
// @Param body body updateBody true "disabled"
// @Success 200 {object} entity.User
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [patch]
func (uc Update) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(w, err)
		return
	}
	caller, _ := usecases.Principal(ctx)
	subject := r.PathValue("subject")
	var body updateBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	if body.Disabled == nil {
		uc.Fail(w, fmt.Errorf("disabled is required"))
		return
	}
	u, err := uc.Users.GetUser(ctx, subject)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if *body.Disabled && (subject == caller.Subject || u.Super) {
		uc.Fail(w, constants.ErrForbidden)
		return
	}
	if err := uc.Users.SetUserDisabled(ctx, subject, *body.Disabled); err != nil {
		uc.Fail(w, err)
		return
	}
	u.Disabled = *body.Disabled
	uc.Respond(w, u)
}
