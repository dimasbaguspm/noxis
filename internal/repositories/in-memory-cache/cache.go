// Package cache is a small TTL key-value store with lazy expiry: Get drops
// expired entries as they are read; no janitor goroutine runs.
//
// ponytail: built speculatively for future memoization — nothing consumes it
// yet. When a shared cache is needed, add it to usecases.Deps (in the handler
// restructure PR) rather than constructing one ad hoc.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value any
	exp   time.Time // zero value = never expires
}

// Cache is a concurrency-safe TTL key-value store.
type Cache struct {
	mu    sync.Mutex
	items map[string]entry
}

// New returns an empty cache.
func New() *Cache { return &Cache{items: make(map[string]entry)} }

// Set stores value under key until ttl elapses. A ttl <= 0 keeps the entry
// until it is deleted (no expiry).
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.items[key] = entry{value: value, exp: exp}
}

// Get returns the stored value and whether it is present. An expired entry is
// dropped and reported absent.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		delete(c.items, key)
		return nil, false
	}
	return e.value, true
}

// Delete removes key whether it is present or not.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
