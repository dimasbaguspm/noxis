package interfaces

import (
	"context"

	"noxis/internal/entity"
	redissvc "noxis/internal/services/redis"
)

// Redis operation shapes live with the module; the result grid is shared.
type (
	Key        = redissvc.Key
	KeyPage    = redissvc.KeyPage
	RedisStats = redissvc.Stats
)

// RedisModule is the console's only route to a Redis server. Writes are
// limited to removing keys: a Redis value cannot be rolled back, so there is
// nothing here that pretends to be a validated change.
type RedisModule interface {
	Module
	Keys(ctx context.Context, c entity.Connection, pattern string, limit int) (KeyPage, error)
	Value(ctx context.Context, c entity.Connection, key string) (Result, error)
	Delete(ctx context.Context, c entity.Connection, keys []string) (int64, error)
	Flush(ctx context.Context, c entity.Connection) error
	Stats(ctx context.Context, c entity.Connection) (RedisStats, error)
}

// Satisfier.
var _ RedisModule = (*redissvc.DB)(nil)
