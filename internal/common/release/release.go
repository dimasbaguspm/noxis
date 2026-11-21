// Package release answers the one question an update notice needs: what is the newest published build,
// and is this binary older than it. The answer comes from an artifact the publish job uploads beside the
// binaries (version.json) rather than from the HTML of a releases page, and it is cached here because a
// deployment asks once an hour, not once per request.
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Info is what a published build says about itself.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	// BuiltAt is the commit timestamp, in RFC 3339, of the build this asset belongs to.
	BuiltAt string `json:"built_at,omitempty"`
}

// Checker fetches the published build at most once per TTL. It never fails the caller: an update check
// that cannot reach GitHub is a missing notice, not an error on the version route.
type Checker struct {
	url    string
	ttl    time.Duration
	client *http.Client

	mu      sync.Mutex
	last    *Info
	checked time.Time
	lastErr error
}

// NewChecker builds a checker for one URL. A zero TTL means "ask every time".
func NewChecker(url string, ttl time.Duration) *Checker {
	return &Checker{
		url: url,
		ttl: ttl,
		// The check is a courtesy: it must never hold a request open for longer than a person notices.
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

// Latest answers the published build, or nil when nothing was published or the fetch failed. The error is
// returned for logging and for the tests; a caller that only wants the notice may ignore it.
func (c *Checker) Latest(ctx context.Context) (*Info, error) {
	if c == nil || c.url == "" {
		return nil, nil //nolint:nilnil // no endpoint means no check, and the caller only wants the notice
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ttl > 0 && !c.checked.IsZero() && time.Since(c.checked) < c.ttl {
		return c.last, c.lastErr
	}
	c.checked = time.Now()

	info, err := c.fetch(ctx)
	c.last, c.lastErr = info, err
	return info, err
}

func (c *Checker) fetch(ctx context.Context) (*Info, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, http.NoBody)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the published build answered %d", response.StatusCode)
	}

	var info Info
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("the published build is not readable: %w", err)
	}
	if info.Commit == "" {
		return nil, errors.New("the published build names no commit")
	}
	return &info, nil
}

// Newer reports whether the published build is a different commit than this one. A build nobody stamped
// (no commit) is never "older": it cannot tell, and a notice that fires on every run is worse than none.
func Newer(published *Info, commit string) bool {
	if published == nil || published.Commit == "" {
		return false
	}
	if commit == "" || commit == "unknown" {
		return false
	}
	return published.Commit != commit
}
