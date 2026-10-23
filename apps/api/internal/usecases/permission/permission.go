// Package permission holds the per-connection RBAC use cases.
package permission

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/constants"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Members returns the known groups and user accounts for permission editing.
type Members struct{ usecases.RegistryDeps }

// @Summary The known groups and user accounts for permission editing
// @Success 200 {object} entity.MemberList
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/config/members [get]
func (uc Members) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	groups, err := uc.Permissions.ListGroupNames(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	users, err := uc.Users.ListUsers(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if groups == nil {
		groups = []string{}
	}
	pickable := make([]entity.Member, 0, len(users))
	for i := range users {
		u := &users[i]
		label := firstNonEmpty(u.Email, u.Username, u.Subject)
		pickable = append(pickable, entity.Member{Subject: u.Subject, Label: label, Role: u.Role})
	}
	uc.Respond(w, entity.MemberList{Groups: groups, Users: pickable})
}

// List lists a connection's permissions.
type List struct{ usecases.RegistryDeps }

// @Summary List a connection's permissions
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.PermissionList
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	// A global admin reads every connection's policy; an instance's own admin
	// reads that one.
	need, err := adminLevel(ctx, entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, mod, r.PathValue("name"), need); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := uc.Permissions.ListPermissions(ctx, r.PathValue("name"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.PermissionList{Permissions: out})
}

// firstNonEmpty returns the first non-empty string, so a label falls back from
// email to username to subject.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// adminLevel is the per-connection level a caller must hold: nothing for a
// global admin (it manages every connection's policy), need for everyone else.
// A policy endpoint resolves it before RawConn so the two gates — global role
// and per-connection grant — stay one call instead of two.
func adminLevel(ctx context.Context, need string) (string, error) {
	p, ok := entity.Principal(ctx)
	if !ok {
		return "", constants.ErrUnauthorized
	}
	if p.IsAdmin() {
		return "", nil
	}
	return need, nil
}

// Set upserts one member's level on a connection.
type Set struct{ usecases.RegistryDeps }

// @Summary Grant a user or group a level on a connection (admin only)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body entity.PermissionGrant true "member, member type and level"
// @Success 200 {object} entity.Permission
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [put]
func (uc Set) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.PermissionGrant
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// granting is the admin surface itself, so only the family is checked here
	if _, err := uc.RawConn(ctx, mod, r.PathValue("name"), ""); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	now := time.Now()
	p := entity.Permission{
		Connection: r.PathValue("name"), Member: body.Member, MemberType: body.MemberType, Level: body.Level,
		ExpiresAt: now.Add(uc.GrantTTL()).Unix(),
		CreatedBy: entity.Actor(ctx),
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := p.Validate(); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Permissions.SetPermission(ctx, p); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, p)
}

// Delete removes a member's permission from a connection. The super admin, a
// global admin and the instance's own admin may call it.
type Delete struct{ usecases.RegistryDeps }

// @Summary Revoke a member's permission on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param member query string true "user subject or group name"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	// The super admin, a global admin and the instance's own admin may revoke.
	need, err := adminLevel(ctx, entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	member := r.URL.Query().Get("member")
	if member == "" {
		uc.Fail(ctx, w, fmt.Errorf("member is required"))
		return
	}
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, mod, r.PathValue("name"), need); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Permissions.DeletePermission(ctx, r.PathValue("name"), member, entity.Actor(ctx)); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}
