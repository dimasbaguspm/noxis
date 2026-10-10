// Package usecases holds the business use cases of the noxis console, one
// file and one Serve per use case. A use case stands alone: it carries only
// the repositories and modules its flow reads, and never references another
// use case — shared steps are package helpers (policy, raw, apply) that
// receive the deps as arguments. Transport concerns live in
// cmd/noxis/server/handlers.
package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ErrForbidden maps to 403: editing the policy is a global-admin action.
var ErrForbidden = errors.New("forbidden: global admin required")

// RequireAdmin gates the management surfaces (connection CRUD, grants). Open
// mode (no principal) is already trusted and passes.
func RequireAdmin(ctx context.Context) error {
	if p, ok := Principal(ctx); !ok || p.Admin {
		return nil
	}
	return ErrForbidden
}

// EffectiveLevel is the strongest level the caller's groups hold on a
// connection; "" when none matches.
func EffectiveLevel(ctx context.Context, grants interfaces.GrantRepository, connection string, groups []string) (string, error) {
	list, err := grants.ListGrants(ctx, connection)
	if err != nil {
		return "", err
	}
	levels := make([]string, 0, len(list))
	for _, g := range list {
		if slices.Contains(groups, g.Group) {
			levels = append(levels, g.Level)
		}
	}
	return entity.StrongestLevel(levels...), nil
}

// Authorize enforces need on a connection. Open mode and global admins pass
// without a per-connection check; everyone else needs a matching grant.
func Authorize(ctx context.Context, grants interfaces.GrantRepository, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := EffectiveLevel(ctx, grants, connection, p.Groups)
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
func CanRead(ctx context.Context, grants interfaces.GrantRepository, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := EffectiveLevel(ctx, grants, connection, p.Groups)
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// readableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func readableConnections(ctx context.Context, conns interfaces.ConnectionRepository, grants interfaces.GrantRepository) (map[string]bool, error) {
	list, err := conns.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		c := &list[i]
		ok, err := CanRead(ctx, grants, c.Name)
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
