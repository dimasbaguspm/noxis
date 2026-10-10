package usecases

import (
	"context"
	"fmt"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ListGrants lists a connection's group grants; a global-admin action.
func (a *App) ListGrants(ctx context.Context, mod interfaces.Module, name string) (out []entity.Grant, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if _, err := a.Raw(ctx, mod, name, entity.LevelRead); err != nil {
		return nil, err
	}
	return a.Grants.ListGrants(ctx, name)
}

// SetGrant grants an OIDC group a level on a connection; a global-admin
// action.
func (a *App) SetGrant(ctx context.Context, mod interfaces.Module, name, group, level string) (out entity.Grant, err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return entity.Grant{}, err
	}
	if _, err := a.Raw(ctx, mod, name, entity.LevelRead); err != nil {
		return entity.Grant{}, err
	}
	g := entity.Grant{Connection: name, Group: group, Level: level}
	if err := g.Validate(); err != nil {
		return entity.Grant{}, err
	}
	if err := a.Grants.SetGrant(ctx, g); err != nil {
		return entity.Grant{}, err
	}
	return g, nil
}

// DeleteGrant revokes a group's grant on a connection; a global-admin action.
func (a *App) DeleteGrant(ctx context.Context, mod interfaces.Module, name, group string) (err error) {
	ctx, end := a.span(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return err
	}
	if group == "" {
		return fmt.Errorf("group is required")
	}
	if _, err := a.Raw(ctx, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return a.Grants.DeleteGrant(ctx, name, group)
}

// Groups returns the known group names for grant editing; a global-admin
// action.
func (a *App) Groups(ctx context.Context) ([]string, error) {
	if err := a.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return a.Grants.ListGroupNames(ctx)
}