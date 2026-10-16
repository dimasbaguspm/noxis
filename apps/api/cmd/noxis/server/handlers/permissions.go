package handlers

import (
	"context"
	"net/http"

	"noxis/cmd/noxis/server/deps"
)

// memberUser is one pickable account: the subject stored on a permission and a
// human label for the console.
type memberUser struct {
	Subject string `json:"subject"`
	Label   string `json:"label"`
}

// @Summary The known groups and user accounts for permission editing
// @Success 200 {object} object{groups=[]string,users=[]memberUser}
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/config/members [get]
func ListConfigMembers(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		groups, users, err := d.Members.Serve(ctx)
		if err != nil {
			return nil, err
		}
		pickable := make([]memberUser, 0, len(users))
		for _, u := range users {
			label := u.Username
			if label == "" {
				label = u.Subject
			}
			pickable = append(pickable, memberUser{Subject: u.Subject, Label: label})
		}
		return map[string]any{"groups": groups, "users": pickable}, nil
	})
}

// permissionBody is the upsert payload: one member at one level.
type permissionBody struct {
	Member     string `json:"member"`
	MemberType string `json:"memberType"`
	Level      string `json:"level"`
}

// @Summary List a connection's permissions
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{permissions=[]entity.Permission}
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [get]
func ListPermissions(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.Modules.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		out, err := d.ListPermissions.Serve(ctx, mod, r.PathValue("name"))
		return map[string]any{"permissions": out}, err
	})
}

// @Summary Grant a user or group a level on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body permissionBody true "member, member type and level"
// @Success 200 {object} entity.Permission
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [put]
func SetPermission(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.Modules.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		var body permissionBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.SetPermission.Serve(ctx, mod, r.PathValue("name"), body.Member, body.MemberType, body.Level)
	})
}

// @Summary Revoke a member's permission on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param member query string true "user subject or group name"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [delete]
func DeletePermission(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handleNoContent(w, r, func(ctx context.Context) error {
		mod, err := d.Modules.Engine(r.PathValue("engine"))
		if err != nil {
			return err
		}
		return d.DeletePermission.Serve(ctx, mod, r.PathValue("name"), r.URL.Query().Get("member"))
	})
}
