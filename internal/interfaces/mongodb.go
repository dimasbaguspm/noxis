package interfaces

import (
	"context"

	"noxis/internal/entity"
	mongosvc "noxis/internal/modules/mongodb"
)

// MongoDB shapes are shared wire types; the result grid is the shared Result.
type (
	MongoDatabase   = entity.Database
	MongoCollection = entity.Collection
	MongoRequest    = entity.MongoRequest
	MongoResult     = entity.Result
)

// MongoModule is the console's only route to a MongoDB server. Reads run
// directly; writes are validated on a scratch collection and then reviewed,
// because a document change has no rollback to undo it.
type MongoModule interface {
	Module
	Databases(ctx context.Context, c entity.Connection) ([]MongoDatabase, error)
	Collections(ctx context.Context, c entity.Connection, database string) ([]MongoCollection, error)
	// Run executes a read request; a write is refused here by design.
	Run(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	// ValidateRequest dry-runs a write against a scratch collection.
	ValidateRequest(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	// Apply executes a write for real, after review.
	Apply(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
}

// Satisfier.
var _ MongoModule = (*mongosvc.DB)(nil)
