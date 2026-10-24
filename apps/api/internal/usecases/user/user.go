// Package user holds the admin user directory use cases. The super admin and promoted admins read it and set account status; only the super admin changes a global role.
package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List is the directory: every account, filtered by a search term and status.
type List struct{ usecases.RegistryDeps }

// @Summary List accounts (admin only)
// @Param q query string false "match subject, email or username"
// @Param status query string false "all, active or banned"
// @Success 200 {object} entity.UserList
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/users [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	users, err := uc.Users.ListUsers(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
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
		if status == entity.StatusActive && !u.Active() {
			continue
		}
		if status == entity.StatusBanned && u.Active() {
			continue
		}
		out = append(out, *u)
	}
	uc.Respond(w, entity.UserList{Users: out})
}

// matches reports whether one account matches a lowercased search term.
func matches(u *entity.User, q string) bool {
	return strings.Contains(strings.ToLower(u.Subject), q) ||
		strings.Contains(strings.ToLower(u.Email), q) ||
		strings.Contains(strings.ToLower(u.Username), q)
}

// Get returns one account, its access (direct and via groups) and its activity.
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
	u, err := uc.Users.GetUser(ctx, r.PathValue("subject"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	access, err := uc.Permissions.ListPermissionsForMembers(ctx, append([]string{u.Subject}, u.Groups...))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	events, err := uc.AccessEvents.ListAccessEvents(ctx, u.Subject, 50)
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
	uc.Respond(w, entity.UserDetail{User: *u, Access: access, Events: events})
}

// Update sets an account's status (active/banned) and, for the super admin only, its global role.
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
	u, err := uc.Users.GetUser(ctx, subject)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// validate the whole patch before mutating, so a bad field cannot leave a half-applied change
	if err := uc.checkPatch(caller, u, body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if subject == caller.Subject {
		uc.Fail(ctx, w, fmt.Errorf("%w: you cannot change your own account", constants.ErrForbidden))
		return
	}
	if u.IsSuper() {
		uc.Fail(ctx, w, fmt.Errorf("%w: the super admin cannot be changed", constants.ErrForbidden))
		return
	}
	if err := uc.applyRole(ctx, caller, u, body.Role); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.applyStatus(ctx, caller, u, body.Status); err != nil {
		uc.Fail(ctx, w, err)
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

// applyStatus bans or re-enables an account. An admin may only act on members, so peers cannot lock each other out; the super admin may act on admins too.
func (uc Update) applyStatus(ctx context.Context, caller entity.Identity, u *entity.User, status *string) error {
	if status == nil {
		return nil
	}
	if !caller.IsSuper() && u.IsAdmin() {
		return fmt.Errorf("%w: only the super admin changes another admin's status", constants.ErrForbidden)
	}
	if *status == u.Status {
		return nil
	}
	if err := uc.Users.SetUserStatus(ctx, u.Subject, *status); err != nil {
		return err
	}
	u.Status = *status
	return nil
}

// checkPatch validates the whole patch against the caller and target, so Serve can mutate without a second field failing halfway through.
func (uc Update) checkPatch(caller entity.Identity, u *entity.User, body entity.UserUpdate) error {
	if body.Role != nil {
		if !caller.IsSuper() {
			return fmt.Errorf("%w: only the super admin changes roles", constants.ErrForbidden)
		}
		if *body.Role != entity.RoleAdmin && *body.Role != entity.RoleMember {
			return fmt.Errorf("role must be %s or %s", entity.RoleAdmin, entity.RoleMember)
		}
	}
	if body.Status != nil {
		if *body.Status != entity.StatusActive && *body.Status != entity.StatusBanned {
			return fmt.Errorf("status must be %s or %s", entity.StatusActive, entity.StatusBanned)
		}
		if !caller.IsSuper() && u.IsAdmin() {
			return fmt.Errorf("%w: only the super admin changes another admin's status", constants.ErrForbidden)
		}
	}
	return nil
}
