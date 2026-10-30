package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// ChangeFlow owns one driver's change lifecycle; use cases authorize, translate, and answer.
type ChangeFlow interface {
	// Driver is the driver whose queue this flow serves.
	Driver() string
	Approve(ctx context.Context, ch *entity.Change, actor string) error
	Reject(ctx context.Context, ch *entity.Change, actor string) error
	Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) error
	// Apply runs an approved change and records the outcome on it.
	Apply(ctx context.Context, ch *entity.Change, actor string) (entity.Change, error)
	// Tick applies every due scheduled change of this driver.
	Tick(ctx context.Context, now int64, actor string) (entity.TickResult, error)
}

// SQLChangeFlow submits SQL batches on top of ChangeFlow.
type SQLChangeFlow interface {
	ChangeFlow
	Submit(ctx context.Context, conn entity.Connection, actor string, ch entity.Change) (entity.Change, error)
}

// MongoChangeFlow submits MongoDB requests on top of ChangeFlow.
type MongoChangeFlow interface {
	ChangeFlow
	Submit(ctx context.Context, conn entity.Connection, actor, notes string, req MongoRequest) (entity.Change, error)
}

// ChangeFlows is every driver's change queue, keyed by driver name, so a driver without a queue has none rather than an empty one.
type ChangeFlows map[string]ChangeFlow
