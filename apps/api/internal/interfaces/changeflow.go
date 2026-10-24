package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// ChangeFlow owns one family's change lifecycle; use cases authorize, translate, and answer.
type ChangeFlow interface {
	// Family is the engine family this flow serves.
	Family() string
	Approve(ctx context.Context, ch *entity.Change, actor string) error
	Reject(ctx context.Context, ch *entity.Change, actor string) error
	Schedule(ctx context.Context, ch *entity.Change, at int64, actor string) error
	// Apply runs an approved change and records the outcome on it.
	Apply(ctx context.Context, ch *entity.Change, actor string) (entity.Change, error)
	// Tick applies every due scheduled change of this family.
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

// ChangeFlows is every family's change queue, resolved together so a use case reaches the one it serves through the registry.
type ChangeFlows struct {
	SQL   SQLChangeFlow
	Mongo MongoChangeFlow
}
