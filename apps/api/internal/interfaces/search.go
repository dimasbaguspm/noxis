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
	Engine
	// Targets lists what the platform indexes, so the console can pick one.
	Targets(ctx context.Context, c entity.Connection) ([]SearchTarget, error)
	// Search runs a read against one target; a write is refused here by design.
	Search(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// ValidateSearchChange proves a write without applying it: the impact the reviewer needs before approving.
	ValidateSearchChange(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// ApplySearch runs a reviewed write for real.
	ApplySearch(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// IsWrite reports whether an op mutates the platform; a read is refused on the change path.
	IsWrite(op string) bool
}
