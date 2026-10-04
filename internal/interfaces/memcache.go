package interfaces

import (
	"context"

	"noxis/internal/entity"
	memcachesvc "noxis/internal/services/memcache"
)

// Redis operation shapes live with the module; the result grid is shared.
type (
	Key           = memcachesvc.Key
	KeyPage       = memcachesvc.KeyPage
	MemcacheStats = memcachesvc.Stats
)

// MemcacheModule is the console's only route to a memory-cache server (Redis
// or Valkey). Writes are limited to removing keys: a cache value cannot be
// rolled back, so there is nothing here that pretends to be a validated
// change.
type MemcacheModule interface {
	Module
	Keys(ctx context.Context, c entity.Connection, pattern string, limit int) (KeyPage, error)
	Value(ctx context.Context, c entity.Connection, key string) (Result, error)
	Delete(ctx context.Context, c entity.Connection, keys []string) (int64, error)
	Flush(ctx context.Context, c entity.Connection) error
	Stats(ctx context.Context, c entity.Connection) (MemcacheStats, error)
}

// Satisfier.
var _ MemcacheModule = (*memcachesvc.DB)(nil)
