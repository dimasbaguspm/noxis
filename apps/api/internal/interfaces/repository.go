// Package interfaces holds the seams between layers (one file per service).
package interfaces

import (
	"context"
	"errors"

	"noxis/internal/entity"
)

// ErrNotFound reports a missing row.
var ErrNotFound = errors.New("not found")

// ErrUnsupported reports an engine name with no module, or an engine that does
// not serve the requested operation. Handlers map it to 404.
var ErrUnsupported = errors.New("unsupported engine")

// ConnectionRepository persists the console's connections.
type ConnectionRepository interface {
	List(ctx context.Context) ([]entity.Connection, error)
	Get(ctx context.Context, name string) (*entity.Connection, error)
	GetByID(ctx context.Context, id string) (*entity.Connection, error)
	Save(ctx context.Context, c entity.Connection) error
	Delete(ctx context.Context, name string) error
}

// PermissionRepository persists the per-connection RBAC policy: which member (a
// user subject or a group) may touch which connection, at what level.
type PermissionRepository interface {
	// ListPermissions returns a connection's permissions, ordered by member.
	ListPermissions(ctx context.Context, connection string) ([]entity.Permission, error)
	// SetPermission upserts one member's level for a connection.
	SetPermission(ctx context.Context, p entity.Permission) error
	// DeletePermission removes one member's permission from a connection.
	DeletePermission(ctx context.Context, connection, member string) error
	// ListGroupNames returns every distinct group that appears in any permission,
	// so the console can suggest groups for new permissions.
	ListGroupNames(ctx context.Context) ([]string, error)
}

// ChangeRepository persists proposed changes (names carry the noun because
// one Store implements both seams).
type ChangeRepository interface {
	ListChanges(ctx context.Context) ([]entity.Change, error)
	GetChange(ctx context.Context, id string) (*entity.Change, error)
	SaveChange(ctx context.Context, c entity.Change) error
	DeleteChange(ctx context.Context, id string) error
}
