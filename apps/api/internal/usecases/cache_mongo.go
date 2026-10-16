package usecases

import "noxis/internal/interfaces"

// The memory-cache and MongoDB-only operations. Connection CRUD is shared
// generically with the other families; these are the family-specific reads and
// (for the cache) removals. A cache value cannot be rolled back, so no key
// mutation is offered as a validated change — writes stop at removing keys.

// CacheKeys browses keys by pattern (bounded scan, with type, TTL and size).
type CacheKeys struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// CacheValue inspects one key's value.
type CacheValue struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// CacheDelete removes the selected keys only.
type CacheDelete struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// CacheFlush empties the connected database (FLUSHDB, never FLUSHALL).
type CacheFlush struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// MongoDatabases lists databases with their sizes.
type MongoDatabases struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// MongoCollections lists a database's collections with their sizes.
type MongoCollections struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// MongoRun runs a read request (find | aggregate | count).
type MongoRun struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}
