package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Redis key shapes are shared wire types.
type (
	Key     = entity.Key
	KeyPage = entity.KeyPage
	// CacheInfo is what a cache server reports about itself.
	CacheInfo = entity.CacheInfo
)

// KeyBrowser is the console's only route to a memory-cache server (redis or valkey): browse, read one value, delete, flush.
type KeyBrowser interface {
	Module
	// Keys browses one page of the keyspace from a cursor; a zero cursor starts the iteration.
	Keys(ctx context.Context, c entity.Connection, pattern string, cursor uint64, limit int) (KeyPage, error)
	Value(ctx context.Context, c entity.Connection, key string) (Result, error)
	Delete(ctx context.Context, c entity.Connection, keys []string) (int64, error)
	Flush(ctx context.Context, c entity.Connection) error
	// Info reports what the server says about itself: memory, eviction, the hit ratio, clients and per-database counts.
	Info(ctx context.Context, c entity.Connection) (CacheInfo, error)
}
