// Package user holds the admin user directory use cases. The super admin and
// promoted admins read it and set account status; only the super admin changes
// a global role.
package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List is the directory: every account, filtered by a search term and status.
type List struct{ usecases.RegistryDeps }

// @Summary List accounts (admin only)
// @Param q query string false "match subject, email or username"
// @Param status query string false "all, active or banned"
// @Success 200 {object} object{users=[]entity.User}
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/users [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireAdmin(ctx); err != nil {
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
		if q != "" && !matches(u, q) {
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

// matches reports whether one account matches a lowercased search term.
func matches(u *entity.User, q string) bool {
	return strings.Contains(strings.ToLower(u.Subject), q) ||
		strings.Contains(strings.ToLower(u.Email), q) ||
		strings.Contains(strings.ToLower(u.Username), q)
}

// Detail is one account with the access it holds and its activity trail.
type Detail struct {
	entity.User
	Access []entity.Permission  `json:"access"`
	Events []entity.AccessEvent `json:"events"`
}

// Get returns one account, its access (direct and via groups) and its activity.
type Get struct{ usecases.RegistryDeps }

// @Summary One account with its access and activity (admin only)
// @Param subject path string true "user subject"
// @Success 200 {object} Detail
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireAdmin(ctx); err != nil {
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

// updateBody is the role payload.
type updateBody struct {
	Role *string `json:"role"`
}

// Update sets an account's global role. Nobody changes their own account and the
// super admin is never demotable.
type Update struct{ usecases.RegistryDeps }

// @Summary Promote or demote an admin (super only)
// @Param subject path string true "user subject"
// @Param body body updateBody true "role"
// @Success 200 {object} entity.User
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/users/{subject} [patch]
func (uc Update) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireAdmin(ctx); err != nil {
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
	if body.Role == nil {
		uc.Fail(w, fmt.Errorf("role is required"))
		return
	}
	u, err := uc.Users.GetUser(ctx, subject)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if subject == caller.Subject {
		uc.Fail(w, fmt.Errorf("%w: you cannot change your own account", constants.ErrForbidden))
		return
	}
	if u.IsSuper() {
		uc.Fail(w, fmt.Errorf("%w: the super admin cannot be changed", constants.ErrForbidden))
		return
	}
	if err := uc.applyRole(ctx, caller, u, body.Role); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, u)
}

// applyRole changes the global role: super only, and never to super.
func (uc Update) applyRole(ctx context.Context, caller entity.Identity, u *entity.User, role *string) error {
	if role == nil {
		return nil
	}
	if !caller.IsSuper() {
		return fmt.Errorf("%w: only the super admin changes roles", constants.ErrForbidden)
	}
	if *role != entity.RoleAdmin && *role != entity.RoleMember {
		return fmt.Errorf("role must be %s or %s", entity.RoleAdmin, entity.RoleMember)
	}
	if *role == u.Role {
		return nil
	}
	if err := uc.Users.SetUserRole(ctx, u.Subject, *role); err != nil {
		return err
	}
	u.Role = *role
	return nil
}
