package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// PermissionRepository persists the per-connection RBAC policy: which member (a
// user subject or a group) may touch which connection, at what level.
type PermissionRepository interface {
	// ListPermissions returns a connection's permissions, ordered by member.
	ListPermissions(ctx context.Context, connection string) ([]entity.Permission, error)
	// SetPermission upserts one member's level for a connection.
	SetPermission(ctx context.Context, p entity.Permission) error
	// DeletePermission removes one member's permission from a connection.
	DeletePermission(ctx context.Context, connection, member string) error
	// ListGroupNames returns every distinct group that appears in any permission.
	ListGroupNames(ctx context.Context) ([]string, error)
}
