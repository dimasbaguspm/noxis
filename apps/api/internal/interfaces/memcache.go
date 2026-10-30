package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Redis key shapes are shared wire types.
type (
	Key     = entity.Key
	KeyPage = entity.KeyPage
)

// KeyBrowser is the console's only route to a memory-cache server (redis or valkey): browse, read one value, delete, flush.
type KeyBrowser interface {
	Module
	Keys(ctx context.Context, c entity.Connection, pattern string, limit int) (KeyPage, error)
	Value(ctx context.Context, c entity.Connection, key string) (Result, error)
	Delete(ctx context.Context, c entity.Connection, keys []string) (int64, error)
	Flush(ctx context.Context, c entity.Connection) error
}
