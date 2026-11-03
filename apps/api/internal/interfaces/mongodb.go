package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// MongoDB shapes are shared wire types; the result grid is the shared Result.
type (
	MongoDatabase   = entity.Database
	MongoCollection = entity.Collection
	MongoRequest    = entity.MongoRequest
	MongoResult     = entity.Result
)

// DocumentBrowser is the console's only route to a document server: browse databases and collections, run a read.
type DocumentBrowser interface {
	Engine
	Databases(ctx context.Context, c entity.Connection) ([]MongoDatabase, error)
	Collections(ctx context.Context, c entity.Connection, database string) ([]MongoCollection, error)
	// Run executes a read request; a write is refused here by design.
	Run(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
}

// MongoChangeEngine is the whole document engine a change queue drives: it proves a request and applies it.
type MongoChangeEngine interface {
	Engine
	// ValidateRequest dry-runs a write against a scratch collection.
	ValidateRequest(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	// ApplyDocument executes a write for real, after review.
	ApplyDocument(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	// IsWrite reports whether an op mutates documents; a read is refused on the change path.
	IsWrite(op string) bool
}

// Satisfier.
