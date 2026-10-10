package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// errForbidden maps to 403: editing the policy is a global-admin action.
var errForbidden = errors.New("forbidden: global admin required")

func (d Deps) grants(ctx context.Context, mod interfaces.Module, name string) (out []entity.Grant, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := d.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := d.raw(ctx, mod, name, entity.LevelRead); err != nil {
		return nil, err
	}
	return d.Grants.ListGrants(ctx, name)
}

func (d Deps) setGrant(ctx context.Context, mod interfaces.Module, name, group, level string) (out entity.Grant, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := d.requireAdmin(ctx); err != nil {
		return entity.Grant{}, err
	}
	if _, err := d.raw(ctx, mod, name, entity.LevelRead); err != nil {
		return entity.Grant{}, err
	}
	g := entity.Grant{Connection: name, Group: group, Level: level}
	if err := g.Validate(); err != nil {
		return entity.Grant{}, err
	}
	if err := d.Grants.SetGrant(ctx, g); err != nil {
		return entity.Grant{}, err
	}
	return g, nil
}

// @Summary The known group names for grant editing
// @Success 200 {object} object{groups=[]string}
// @Failure 403 {object} map[string]string
// @Router /v1/config/groups [get]
func ListConfigGroups(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.requireAdmin(ctx); err != nil {
			return nil, err
		}
		groups, err := d.Grants.ListGroupNames(ctx)
		if err != nil {
			return nil, err
		}
		if groups == nil {
			groups = []string{}
		}
		return map[string]any{"groups": groups}, nil
	})
}

func (d Deps) deleteGrant(ctx context.Context, mod interfaces.Module, name, group string) (err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := d.requireAdmin(ctx); err != nil {
		return err
	}
	if group == "" {
		return fmt.Errorf("group is required")
	}
	if _, err := d.raw(ctx, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return d.Grants.DeleteGrant(ctx, name, group)
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
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/grants [get]
func ListGrants(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		out, err := d.grants(ctx, mod, r.PathValue("name"))
		return map[string]any{"grants": out}, err
	})
}

// @Summary Grant an OIDC group a level on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body grantBody true "group and level"
// @Success 200 {object} entity.Grant
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/grants [put]
func SetGrant(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		var body grantBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.setGrant(ctx, mod, r.PathValue("name"), body.Group, body.Level)
	})
}

// @Summary Revoke a group's grant on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param group query string true "OIDC group name"
// @Success 204
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/grants [delete]
func DeleteGrant(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		mod, err := d.engine(r)
		if err != nil {
			return err
		}
		return d.deleteGrant(ctx, mod, r.PathValue("name"), r.URL.Query().Get("group"))
	})
}
