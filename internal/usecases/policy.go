package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// ErrForbidden maps to 403: editing the policy is a global-admin action.
var ErrForbidden = errors.New("forbidden: global admin required")

// RequireAdmin gates the management surfaces (connection CRUD, grants). Open
// mode (no principal) is already trusted and passes.
func (a *App) RequireAdmin(ctx context.Context) error {
	if p, ok := Principal(ctx); !ok || p.Admin {
		return nil
	}
	return ErrForbidden
}

// EffectiveLevel is the strongest level the caller's groups hold on a
// connection; "" when none matches.
func (a *App) EffectiveLevel(ctx context.Context, connection string, groups []string) (string, error) {
	grants, err := a.Grants.ListGrants(ctx, connection)
	if err != nil {
		return "", err
	}
	levels := make([]string, 0, len(grants))
	for _, g := range grants {
		if slices.Contains(groups, g.Group) {
			levels = append(levels, g.Level)
		}
	}
	return entity.StrongestLevel(levels...), nil
}

// Authorize enforces need on a connection. Open mode and global admins pass
// without a per-connection check; everyone else needs a matching grant.
func (a *App) Authorize(ctx context.Context, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := a.EffectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, need) {
		return fmt.Errorf("%w: %s requires %s on connection %q", ErrForbidden, p.Subject, need, connection)
	}
	return nil
}

// CanRead reports whether the caller may read a connection; used to hide
// connections from listings rather than failing the whole list.
func (a *App) CanRead(ctx context.Context, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := a.EffectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// readableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func (a *App) readableConnections(ctx context.Context) (map[string]bool, error) {
	list, err := a.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		c := &list[i]
		ok, err := a.CanRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		out[c.Name] = ok
	}
	return out, nil
}

// OpLevel is the level a direct SQL op requires. Everything that is not a
// read is a write (ANALYZE mutates planner statistics).
func OpLevel(operation string) string {
	if operation == constants.OpQuery {
		return entity.LevelRead
	}
	return entity.LevelWrite
}