package controller

import (
	"context"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// requireAdmin gates the management surfaces (connection CRUD, grants). Open
// mode (no principal) is already trusted and passes.
func (d Deps) requireAdmin(ctx context.Context) error {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	return errForbidden
}

// effectiveLevel is the strongest level the caller's groups hold on a
// connection; "" when none matches.
func (d Deps) effectiveLevel(ctx context.Context, connection string, groups []string) (string, error) {
	grants, err := d.Grants.ListGrants(ctx, connection)
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

// authorize enforces need on a connection. Open mode and global admins pass
// without a per-connection check; everyone else needs a matching grant.
func (d Deps) authorize(ctx context.Context, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := d.effectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, need) {
		return fmt.Errorf("%w: %s requires %s on connection %q", errForbidden, p.Subject, need, connection)
	}
	return nil
}

// canRead reports whether the caller may read a connection; used to hide
// connections from listings rather than failing the whole list.
func (d Deps) canRead(ctx context.Context, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := d.effectiveLevel(ctx, connection, p.Groups)
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// readableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func (d Deps) readableConnections(ctx context.Context) (map[string]bool, error) {
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for _, c := range list {
		ok, err := d.canRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		out[c.Name] = ok
	}
	return out, nil
}

// opLevel is the level a direct SQL op requires. Everything that is not a
// read is a write (ANALYZE mutates planner statistics).
func opLevel(operation string) string {
	if operation == constants.OpQuery {
		return entity.LevelRead
	}
	return entity.LevelWrite
}
