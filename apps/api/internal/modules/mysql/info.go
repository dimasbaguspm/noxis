package mysql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"sync"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// versionQuery asks the server for its version: plain SQL rather than a dialect field, so it lives beside the profile.
const versionQuery = `SELECT VERSION()`

// capabilities names what this profile lets the console prove, in the operator's words. It is derived from the profile and never from a typed version: a wrong number would silently pick a weaker proof.
func (d dialect) capabilities() []string {
	caps := make([]string, 0, 4)
	if d.txn == txnAlways {
		caps = append(caps, "ddl rolled back in a transaction")
	} else {
		caps = append(caps, "ddl proved in a scratch schema", "qualified ddl proved in scratch")
	}
	if d.explain != "" {
		caps = append(caps, "planner row estimates")
	}
	if d.commentsQuery != "" {
		caps = append(caps, "table comments")
	}
	return caps
}

// infoCache remembers what a server said, keyed by the connection it answered for, so the version is read once per connection instead of on every page load.
type infoCache struct {
	mu sync.Mutex
	by map[string]entity.ServerInfo
}

func (c *infoCache) get(key string) (entity.ServerInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, ok := c.by[key]
	return info, ok
}

func (c *infoCache) put(key string, info entity.ServerInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.by == nil {
		c.by = map[string]entity.ServerInfo{}
	}
	c.by[key] = info
}

// ServerInfo reports the engine, the version the server gave, and what that profile can prove. A server that will not answer is an error: a version guessed from a default is exactly the lie this feature exists to avoid.
func (db *DB) ServerInfo(ctx context.Context, c entity.Connection) (info entity.ServerInfo, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	d, err := dialectOf(c.Driver)
	if err != nil {
		return entity.ServerInfo{}, err
	}
	key := c.Name + "|" + d.name + "|" + c.Host + "|" + c.Database
	if cached, ok := db.info.get(key); ok {
		return cached, nil
	}
	info = entity.ServerInfo{Engine: d.name, Capabilities: d.capabilities()}
	_, err = withDB(c, func(sqlDB *stdsql.DB) (struct{}, error) {
		if qerr := sqlDB.QueryRowContext(ctx, versionQuery).Scan(&info.Version); qerr != nil {
			return struct{}{}, fmt.Errorf("read server version: %w", qerr)
		}
		return struct{}{}, nil
	})
	if err != nil {
		return entity.ServerInfo{}, err
	}
	db.info.put(key, info)
	return info, nil
}
