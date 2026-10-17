// Package usecases holds the business use cases of the noxis console, one
// file per use case. The domain work is a Run core; its Serve(w, r) method
// owns that route's HTTP and there is no separate handler package. A use case
// stands alone: it carries only the repositories and modules its flow reads,
// and never references another use case — shared steps are package helpers
// (policy, raw, apply) that receive the deps as arguments.
package usecases

import (
	"context"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// RequireAdmin gates the management surfaces (connection CRUD, permissions).
// The middleware guarantees a principal; a missing one is denied, not trusted.
func RequireAdmin(ctx context.Context) error {
	if p, ok := Principal(ctx); ok && p.Admin {
		return nil
	}
	return constants.ErrForbidden
}

// identities is the caller's match set: the subject plus every group it belongs
// to. A permission matches when its member is in this set.
func identities(p entity.Identity) []string {
	return append([]string{p.Subject}, p.Groups...)
}

// EffectiveLevel is the strongest level the caller's identities hold on a
// connection; "" when none matches.
func EffectiveLevel(ctx context.Context, perms interfaces.PermissionRepository, connection string, identities []string) (string, error) {
	list, err := perms.ListPermissions(ctx, connection)
	if err != nil {
		return "", err
	}
	levels := make([]string, 0, len(list))
	for _, p := range list {
		if slices.Contains(identities, p.Member) {
			levels = append(levels, p.Level)
		}
	}
	return entity.StrongestLevel(levels...), nil
}

// Authorize enforces need on a connection. Global admins pass without a
// per-connection check; everyone else needs a matching permission.
func Authorize(ctx context.Context, perms interfaces.PermissionRepository, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := EffectiveLevel(ctx, perms, connection, identities(p))
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, need) {
		return fmt.Errorf("%w: %s requires %s on connection %q", constants.ErrForbidden, p.Subject, need, connection)
	}
	return nil
}

// CanRead reports whether the caller may read a connection; used to hide
// connections from listings rather than failing the whole list.
func CanRead(ctx context.Context, perms interfaces.PermissionRepository, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := EffectiveLevel(ctx, perms, connection, identities(p))
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// readableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func readableConnections(ctx context.Context, conns interfaces.ConnectionRepository, perms interfaces.PermissionRepository) (map[string]bool, error) {
	list, err := conns.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		c := &list[i]
		ok, err := CanRead(ctx, perms, c.Name)
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
