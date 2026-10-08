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
	Save(ctx context.Context, c entity.Connection) error
	Delete(ctx context.Context, name string) error
}

// GrantRepository persists the per-connection RBAC policy: which OIDC group
// may touch which connection, at what level.
type GrantRepository interface {
	// ListGrants returns a connection's grants, ordered by group.
	ListGrants(ctx context.Context, connection string) ([]entity.Grant, error)
	// SetGrant upserts one group's level for a connection.
	SetGrant(ctx context.Context, g entity.Grant) error
	// DeleteGrant removes one group's grant from a connection.
	DeleteGrant(ctx context.Context, connection, group string) error
}

// ChangeRepository persists proposed changes (names carry the noun because
// one Store implements both seams).
type ChangeRepository interface {
	ListChanges(ctx context.Context) ([]entity.Change, error)
	GetChange(ctx context.Context, id string) (*entity.Change, error)
	SaveChange(ctx context.Context, c entity.Change) error
	DeleteChange(ctx context.Context, id string) error
}
