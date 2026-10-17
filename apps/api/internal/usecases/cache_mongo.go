package usecases

// The memory-cache and MongoDB-only operations. Connection CRUD is shared
// generically with the other families; these are the family-specific reads and
// (for the cache) removals. A cache value cannot be rolled back, so no key
// mutation is offered as a validated change — writes stop at removing keys.

// CacheKeys browses keys by pattern (bounded scan, with type, TTL and size).
type CacheKeys struct {
	RegistryDeps
}

// CacheValue inspects one key's value.
type CacheValue struct {
	RegistryDeps
}

// CacheDelete removes the selected keys only.
type CacheDelete struct {
	RegistryDeps
}

// CacheFlush empties the connected database (FLUSHDB, never FLUSHALL).
type CacheFlush struct {
	RegistryDeps
}

// MongoDatabases lists databases with their sizes.
type MongoDatabases struct {
	RegistryDeps
}

// MongoCollections lists a database's collections with their sizes.
type MongoCollections struct {
	RegistryDeps
}

// MongoRun runs a read request (find | aggregate | count).
type MongoRun struct {
	RegistryDeps
}
