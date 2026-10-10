package handlers

import (
	"context"
	"net/http"
	"noxis/internal/usecases"
)

// @Summary The known group names for grant editing
// @Success 200 {object} object{groups=[]string}
// @Failure 403 {object} entity.ErrorResponse
// @Router /v1/config/groups [get]
func ListConfigGroups(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		groups, err := d.Grants.Groups(ctx)
		if err != nil {
			return nil, err
		}
		if groups == nil {
			groups = []string{}
		}
		return map[string]any{"groups": groups}, nil
	})
}

// grantBody is the upsert payload: one group at one level.
type grantBody struct {
	Group string `json:"group"`
	Level string `json:"level"`
}

// @Summary List a connection's group grants
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{grants=[]entity.Grant}
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/grants [get]
func ListGrants(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.Engines.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		out, err := d.Grants.ListGrants(ctx, mod, r.PathValue("name"))
		return map[string]any{"grants": out}, err
	})
}

// @Summary Grant an OIDC group a level on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body grantBody true "group and level"
// @Success 200 {object} entity.Grant
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/grants [put]
func SetGrant(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.Engines.Engine(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		var body grantBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.Grants.SetGrant(ctx, mod, r.PathValue("name"), body.Group, body.Level)
	})
}

// @Summary Revoke a group's grant on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param group query string true "OIDC group name"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/grants [delete]
func DeleteGrant(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handleNoContent(w, r, func(ctx context.Context) error {
		mod, err := d.Engines.Engine(r.PathValue("engine"))
		if err != nil {
			return err
		}
		return d.Grants.DeleteGrant(ctx, mod, r.PathValue("name"), r.URL.Query().Get("group"))
	})
}
