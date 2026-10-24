// Package usecases is the framework every use case embeds.
package usecases

import (
	"context"
	"fmt"
	"slices"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// RequireSuper gates the root-only surfaces: connection lifecycle, promotions and instance credentials.
func (d RegistryDeps) RequireSuper(ctx context.Context) error {
	if p, ok := entity.Principal(ctx); ok && p.IsSuper() {
		return nil
	}
	logging.WarnWithCtx(ctx, "super admin required", "gate", "super")
	return constants.ErrForbidden
}

// RequireAdmin gates the instance-management surfaces: the user directory, account status and the per-connection grant policy.
func (d RegistryDeps) RequireAdmin(ctx context.Context) error {
	if p, ok := entity.Principal(ctx); ok && p.IsAdmin() {
		return nil
	}
	logging.WarnWithCtx(ctx, "admin role required", "gate", "admin")
	return constants.ErrForbidden
}

// identities is the caller's match set: the subject plus every group it belongs to. A permission matches when its member is in this set.
func identities(p entity.Identity) []string {
	return append([]string{p.Subject}, p.Groups...)
}

// EffectiveGrant is the strongest non-expired level the caller holds on a connection, with the expiry of the grant that supplies it. "" when none.
func (d RegistryDeps) EffectiveGrant(ctx context.Context, connection string, p entity.Identity) (level string, expiresAt int64, err error) {
	list, err := d.Repositories.Permissions.ListPermissions(ctx, connection)
	if err != nil {
		return "", 0, err
	}
	ids := identities(p)
	now := time.Now().Unix()
	for i := range list {
		grant := &list[i]
		if grant.Expired(now) || !slices.Contains(ids, grant.Member) {
			continue
		}
		if level == "" || entity.LevelAllows(grant.Level, level) {
			level, expiresAt = grant.Level, grant.ExpiresAt
		}
	}
	return level, expiresAt, nil
}

// EffectiveLevel is EffectiveGrant without the expiry.
func (d RegistryDeps) EffectiveLevel(ctx context.Context, connection string, p entity.Identity) (string, error) {
	level, _, err := d.EffectiveGrant(ctx, connection, p)
	return level, err
}

// Authorize enforces need on a connection. The super admin passes without a per-connection check; everyone else — a promoted admin included — needs a matching, unexpired permission.
func (d RegistryDeps) Authorize(ctx context.Context, connection, need string) error {
	if need == "" {
		return nil
	}
	p, ok := entity.Principal(ctx)
	if !ok || p.IsSuper() {
		return nil
	}
	level, err := d.EffectiveLevel(ctx, connection, p)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, need) {
		logging.WarnWithCtx(ctx, "access denied",
			"connection", connection, "need", need, "have", level, "subject", p.Subject)
		return fmt.Errorf("%w: %s requires %s on connection %q", constants.ErrForbidden, p.Subject, need, connection)
	}
	return nil
}

// CanRead reports whether the caller may read a connection; used to hide connections from listings rather than failing the whole list.
func (d RegistryDeps) CanRead(ctx context.Context, connection string) (bool, error) {
	p, ok := entity.Principal(ctx)
	if !ok || p.IsSuper() {
		return true, nil
	}
	level, err := d.EffectiveLevel(ctx, connection, p)
	if err != nil {
		return false, err
	}
	return entity.LevelAllows(level, entity.LevelRead), nil
}

// ReadableConnections is the set of connection names the caller may read, so a change list can drop rows for connections the caller cannot see.
func (d RegistryDeps) ReadableConnections(ctx context.Context) (map[string]bool, error) {
	list, err := d.Repositories.Connections.List(ctx)
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

// OpLevel is the level a direct SQL op requires.
func (d RegistryDeps) OpLevel(operation string) string {
	if operation == constants.OpQuery {
		return entity.LevelRead
	}
	return entity.LevelWrite
}
