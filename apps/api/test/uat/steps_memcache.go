package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// Memory-cache scenarios browse Redis/Valkey; server CLIs seed keys because noXis has no set-key endpoint.

// redisKey is one entry of a browse.
type redisKey struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	TTL     int64  `json:"ttlMs"`
	HasByte int64  `json:"sizeBytes"`
}

// redisPage is a browse result.
type redisPage struct {
	Keys      []redisKey `json:"keys"`
	Total     int64      `json:"total"`
	Truncated bool       `json:"truncated"`
	Cursor    uint64     `json:"cursor"`
	Pattern   string     `json:"pattern"`
}

// cacheInfoBody is what a cache server says about itself, only the fields UAT asserts on.
type cacheInfoBody struct {
	UsedMemoryBytes int64 `json:"usedMemoryBytes"`
	Databases       []struct {
		Keys int64 `json:"keys"`
	} `json:"databases"`
}

func (w *world) memcacheSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a redis connection named "([^"]*)"$`, w.registerRedis)
	sc.Step(`^a valkey connection named "([^"]*)"$`, w.registerValkey)
	sc.Step(`^a dragonfly connection named "([^"]*)"$`, w.registerDragonfly)
	sc.Step("the redis server answers", w.redisAnswers)
	sc.Step(`^the redis key "([^"]*)" holds "([^"]*)"$`, w.redisSeedString)
	sc.Step(`^the redis key "([^"]*)" lives for (\d+) seconds$`, w.redisSeedTTL)
	sc.Step(`^the redis list "([^"]*)" contains "([^"]*)"$`, w.redisSeedList)
	sc.Step(`^I browse redis keys matching "([^"]*)"$`, w.redisBrowse)
	sc.Step(`^I browse redis keys matching "([^"]*)" with a limit of (\d+)$`, w.redisBrowseLimited)
	sc.Step("the browse is truncated with a cursor", w.redisBrowseTruncated)
	sc.Step("I continue the browse from that cursor", w.redisBrowseMore)
	sc.Step(`^the browse returns (\d+) key\(s\)$`, w.redisBrowseCount)
	sc.Step(`^I try to browse redis keys matching "([^"]*)"$`, w.redisBrowseToleratingFailure)
	sc.Step(`^the browse finds the key "([^"]*)"$`, w.redisBrowseFinds)
	sc.Step(`^the browse does not find the key "([^"]*)"$`, w.redisBrowseMisses)
	sc.Step(`^the key "([^"]*)" reports a ttl$`, w.redisKeyHasTTL)
	sc.Step(`^I inspect the redis key "([^"]*)"$`, w.redisInspect)
	sc.Step(`^the value grid shows "([^"]*)"$`, w.redisValueShows)
	sc.Step(`^I delete the redis keys "([^"]*)"$`, w.redisDelete)
	sc.Step(`^I empty the redis database as "([^"]*)"$`, w.redisFlush)
	sc.Step("the redis flush is rejected", w.redisFlushRejected)
	sc.Step("the redis database is empty", w.redisEmpty)
	sc.Step("I read the cache server's stats", w.redisStats)
	sc.Step("the cache stats report a used memory and at least one key", w.redisStatsSane)
}

// redisStats reads what the server says about itself, which is the only place memory and eviction are reported.
func (w *world) redisStats(ctx context.Context) error {
	var out cacheInfoBody
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn+"/stats"), nil, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("cache stats: %d", w.code)
	}
	w.cacheInfo = out
	return nil
}

// redisStatsSane proves the report carries numbers rather than an empty shape: memory used, and keys the server counts.
func (w *world) redisStatsSane(ctx context.Context) error {
	if w.cacheInfo.UsedMemoryBytes <= 0 {
		return fmt.Errorf("used memory %d, want a positive number", w.cacheInfo.UsedMemoryBytes)
	}
	keys := int64(0)
	for _, database := range w.cacheInfo.Databases {
		keys += database.Keys
	}
	if keys < 1 {
		return fmt.Errorf("databases %+v report no keys", w.cacheInfo.Databases)
	}
	return nil
}

func (w *world) registerRedis(ctx context.Context, name string) error {
	return w.registerCache(ctx, name, "redis")
}

// registerValkey registers the compose valkey service: same protocol, its own driver name, and it must be served by the same module.
func (w *world) registerValkey(ctx context.Context, name string) error {
	return w.registerCache(ctx, name, "valkey")
}

// registerDragonfly registers the compose dragonfly service: redis protocol, its own driver name.
func (w *world) registerDragonfly(ctx context.Context, name string) error {
	return w.registerCache(ctx, name, "dragonfly")
}

// registerCache builds the memcache connection from the target the compose file declares.
func (w *world) registerCache(ctx context.Context, name, key string) error {
	t := targets[key]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port, "database": t.Database,
	}
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register %s connection: %d", key, w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, key
	return nil
}

func (w *world) redisAnswers(ctx context.Context) error {
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/test"), map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("redis ping: %d", w.code)
	}
	return nil
}

func (w *world) redisSeedString(ctx context.Context, key, value string) error {
	return redisSeed(w.service(), "set", key, value)
}

func (w *world) redisSeedTTL(ctx context.Context, key string, seconds int) error {
	return redisSeed(w.service(), "expire", key, fmt.Sprint(seconds))
}

func (w *world) redisSeedList(ctx context.Context, key, value string) error {
	return redisSeed(w.service(), "rpush", key, value)
}

// service is the compose service a key-value scenario seeds into.
func (w *world) service() string { return w.host }

func (w *world) redisBrowse(ctx context.Context, pattern string) error {
	w.page = redisPage{}
	path := fmt.Sprintf(w.enginePath("/connections/%s/keys?pattern=%s"), w.conn, pattern)
	if err := w.api(http.MethodGet, path, nil, &w.page); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("browse: %d", w.code)
	}
	return nil
}

// redisBrowseToleratingFailure browses without insisting on success, so a rejected request can be asserted next.
// redisBrowseLimited browses one page small enough to leave the scan unfinished, so the cursor can be exercised.
func (w *world) redisBrowseLimited(ctx context.Context, pattern string, limit int) error {
	w.page = redisPage{}
	path := fmt.Sprintf(w.enginePath("/connections/%s/keys?pattern=%s&limit=%d"), w.conn, pattern, limit)
	if err := w.api(http.MethodGet, path, nil, &w.page); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("browse: %d", w.code)
	}
	return nil
}

// redisBrowseTruncated proves the server said there is more and handed back where it stopped.
func (w *world) redisBrowseTruncated(ctx context.Context) error {
	if !w.page.Truncated || w.page.Cursor == 0 {
		return fmt.Errorf("page truncated=%v cursor=%d, want more to read", w.page.Truncated, w.page.Cursor)
	}
	return nil
}

// redisBrowseMore continues from the cursor the last page returned, which is what the page's load more does.
func (w *world) redisBrowseMore(ctx context.Context) error {
	pattern, cursor := w.page.Pattern, w.page.Cursor
	next := redisPage{}
	path := fmt.Sprintf(w.enginePath("/connections/%s/keys?pattern=%s&cursor=%d"), w.conn, pattern, cursor)
	if err := w.api(http.MethodGet, path, nil, &next); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("continue browse: %d", w.code)
	}
	w.page = next
	return nil
}

// redisBrowseCount asserts how many keys one page held.
func (w *world) redisBrowseCount(ctx context.Context, want int) error {
	if len(w.page.Keys) != want {
		return fmt.Errorf("page held %d key(s), want %d", len(w.page.Keys), want)
	}
	return nil
}

func (w *world) redisBrowseToleratingFailure(ctx context.Context, pattern string) error {
	w.page = redisPage{}
	path := fmt.Sprintf(w.enginePath("/connections/%s/keys?pattern=%s"), w.conn, pattern)
	return w.api(http.MethodGet, path, nil, &w.page)
}

func (w *world) redisBrowseFinds(ctx context.Context, key string) error {
	for _, k := range w.page.Keys {
		if k.Name == key {
			w.key = k
			return nil
		}
	}
	return fmt.Errorf("key %q not in %d browsed key(s)", key, len(w.page.Keys))
}

func (w *world) redisBrowseMisses(ctx context.Context, key string) error {
	for _, k := range w.page.Keys {
		if k.Name == key {
			return fmt.Errorf("key %q survived the restore", key)
		}
	}
	return nil
}

func (w *world) redisKeyHasTTL(ctx context.Context, key string) error {
	if w.key.Name != key {
		return fmt.Errorf("key under test is %q, not %q", w.key.Name, key)
	}
	if w.key.TTL <= 0 {
		return fmt.Errorf("key %q ttl=%d, want a positive ttl", key, w.key.TTL)
	}
	return nil
}

func (w *world) redisInspect(ctx context.Context, key string) error {
	w.result = resultBody{}
	path := fmt.Sprintf(w.enginePath("/connections/%s/value?key=%s"), w.conn, key)
	if err := w.api(http.MethodGet, path, nil, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("inspect %s: %d", key, w.code)
	}
	return nil
}

func (w *world) redisValueShows(ctx context.Context, want string) error {
	for _, row := range w.result.Rows {
		for _, cell := range row {
			if strings.Contains(cell, want) {
				return nil
			}
		}
	}
	return fmt.Errorf("no cell contains %q in %v", want, w.result.Rows)
}

func (w *world) redisDelete(ctx context.Context, keys string) error {
	body := map[string]any{"keys": strings.Split(keys, ",")}
	path := w.enginePath("/connections/" + w.conn + "/keys")
	var out struct {
		Deleted int64 `json:"deleted"`
	}
	if err := w.api(http.MethodDelete, path, body, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK || out.Deleted != int64(len(strings.Split(keys, ","))) {
		return fmt.Errorf("delete %s: %d deleted=%d", keys, w.code, out.Deleted)
	}
	return nil
}

func (w *world) redisFlush(ctx context.Context, confirm string) error {
	path := w.enginePath("/connections/" + w.conn + "/flush")
	if err := w.api(http.MethodPost, path, map[string]string{"confirm": confirm}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("flush: %d", w.code)
	}
	return nil
}

// redisFlushRejected attempts the empty with a wrong confirmation and insists it is refused: the guard is the whole point of the confirmation.
func (w *world) redisFlushRejected(ctx context.Context) error {
	path := w.enginePath("/connections/" + w.conn + "/flush")
	if err := w.api(http.MethodPost, path, map[string]string{"confirm": "not-the-name"}, nil); err != nil {
		return err
	}
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("flush with a wrong confirmation: %d, want 400", w.code)
	}
	return nil
}

func (w *world) redisEmpty(ctx context.Context) error {
	if err := w.redisBrowse(ctx, "*"); err != nil {
		return err
	}
	if len(w.page.Keys) != 0 || w.page.Total != 0 {
		return fmt.Errorf("database still holds %d key(s)", w.page.Total)
	}
	return nil
}
