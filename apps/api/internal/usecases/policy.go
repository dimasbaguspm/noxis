// Package usecases is the framework every use case embeds. RegistryDeps carries
// the repositories and the module registry, and its methods are the shared
// steps — RawConn, Authorize, RequireAdmin, CanRead, Respond, Fail — so a use
// case reaches them statefully. Concrete use cases live in the domain
// subpackages and never reference each other.
package usecases

import (
	"context"
	"fmt"
	"slices"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// RequireAdmin gates the management surfaces (connection CRUD, permissions).
// The middleware guarantees a principal; a missing one is denied, not trusted.
func (d RegistryDeps) RequireAdmin(ctx context.Context) error {
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
func (d RegistryDeps) EffectiveLevel(ctx context.Context, connection string, ids []string) (string, error) {
	list, err := d.Permissions.ListPermissions(ctx, connection)
	if err != nil {
		return "", err
	}
	levels := make([]string, 0, len(list))
	for _, p := range list {
		if slices.Contains(ids, p.Member) {
			levels = append(levels, p.Level)
		}
	}
	return entity.StrongestLevel(levels...), nil
}

// Authorize enforces need on a connection. Global admins pass without a
// per-connection check; everyone else needs a matching permission.
func (d RegistryDeps) Authorize(ctx context.Context, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return nil
	}
	level, err := d.EffectiveLevel(ctx, connection, identities(p))
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
func (d RegistryDeps) CanRead(ctx context.Context, connection string) (bool, error) {
	p, ok := Principal(ctx)
	if !ok || p.Admin {
		return true, nil
	}
	level, err := d.EffectiveLevel(ctx, connection, identities(p))
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// ReadableConnections is the set of connection names the caller may read, so
// a change list can drop rows for connections the caller cannot see.
func (d RegistryDeps) ReadableConnections(ctx context.Context) (map[string]bool, error) {
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		c := &list[i]
		ok, err := d.CanRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		out[c.Name] = ok
	}
	return out, nil
}

// OpLevel is the level a direct SQL op requires. Everything that is not a
// read is a write (ANALYZE mutates planner statistics).
func (d RegistryDeps) OpLevel(operation string) string {
	if operation == constants.OpQuery {
		return entity.LevelRead
	}
	return entity.LevelWrite
}
