package interfaces

import (
	"context"

	"noxis/internal/entity"
	mongosvc "noxis/internal/services/mongodb"
)

// MongoDB operation shapes live with the module; the result grid is shared.
type (
	MongoDatabase   = mongosvc.Database
	MongoCollection = mongosvc.Collection
	MongoRequest    = mongosvc.Request
	MongoStats      = mongosvc.Stats
)

// MongoModule is the console's only route to a MongoDB server. Requests run
// directly: a document change has no generic rollback, so the guards are the
// op allowlist and the refusal to update or delete without a filter.
type MongoModule interface {
	Module
	Databases(ctx context.Context, c entity.Connection) ([]MongoDatabase, error)
	Collections(ctx context.Context, c entity.Connection, database string) ([]MongoCollection, error)
	Run(ctx context.Context, c entity.Connection, req MongoRequest) (Result, error)
	Stats(ctx context.Context, c entity.Connection, database, collection string) (MongoStats, error)
}

// Satisfier.
var _ MongoModule = (*mongosvc.DB)(nil)
