package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Search shapes are shared wire types.
type (
	SearchTarget  = entity.SearchTarget
	SearchRequest = entity.SearchRequest
)

// SearchEngine is the whole search platform a queue drives: it browses what the platform indexes, runs a query, and
// proves a change before anyone approves it. A search platform is neither SQL nor a document store, so it gets its own
// seam rather than a borrowed one.
type SearchEngine interface {
	Module
	// Targets lists what the platform indexes, so the console can pick one.
	Targets(ctx context.Context, c entity.Connection) ([]SearchTarget, error)
	// Search runs a read against one target; a write is refused here by design.
	Search(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// ValidateChange proves a write without applying it: the impact the reviewer needs before approving.
	ValidateChange(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// Apply runs a reviewed write for real.
	Apply(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// IsWrite reports whether an op mutates the platform; a read is refused on the change path.
	IsWrite(op string) bool
}

// SearchChangeFlow submits search-platform changes on top of ChangeFlow.
type SearchChangeFlow interface {
	ChangeFlow
	Submit(ctx context.Context, conn entity.Connection, actor, notes string, req SearchRequest) (entity.Change, error)
}
