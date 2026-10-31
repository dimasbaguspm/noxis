package entity

// CacheInfo is what a cache server says about itself: the numbers an operator opens the console for, rather than a table
// of keys with no sense of how full the server is.
type CacheInfo struct {
	// Version is the engine's own version string, so a page can say what it is talking to.
	Version string `json:"version"`
	// UptimeSeconds is how long the server has been up.
	UptimeSeconds int64 `json:"uptimeSeconds"`
	// UsedMemoryBytes, PeakMemoryBytes and MaxMemoryBytes are what the server reports about its memory; a zero maximum
	// means no limit is configured, which is a fact rather than a missing value.
	UsedMemoryBytes int64 `json:"usedMemoryBytes"`
	PeakMemoryBytes int64 `json:"peakMemoryBytes"`
	MaxMemoryBytes  int64 `json:"maxMemoryBytes"`
	// EvictedKeys and ExpiredKeys are cumulative counts since boot.
	EvictedKeys int64 `json:"evictedKeys"`
	ExpiredKeys int64 `json:"expiredKeys"`
	// KeyspaceHits and KeyspaceMisses give the cache's hit ratio.
	KeyspaceHits   int64 `json:"keyspaceHits"`
	KeyspaceMisses int64 `json:"keyspaceMisses"`
	// ConnectedClients and OpsPerSecond are the server's current load.
	ConnectedClients int64 `json:"connectedClients"`
	OpsPerSecond     int64 `json:"opsPerSecond"`
	// Databases are the logical databases the server reports, with how many keys each holds.
	Databases []CacheDatabase `json:"databases"`
}

// CacheDatabase is one logical database and what the server reports about it.
type CacheDatabase struct {
	Name    string `json:"name"`
	Keys    int64  `json:"keys"`
	Expires int64  `json:"expires"`
}
