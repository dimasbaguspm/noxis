// Package http is the shared client for the master REST API. Every request
// carries the standard headers (content type, accept, user agent); callers
// only pass method, path, and payloads.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "noxis/0.1.0"

type Client struct {
	base string
	hc   *http.Client
}

// NewClient returns a client for the master at base (NOXIS_MASTER).
func NewClient(base string) *Client {
	return &Client{base: base, hc: &http.Client{Timeout: 10 * time.Second}}
}

// Do sends a request. body (may be nil) is JSON-encoded; out (may be nil)
// decodes the JSON response. Non-2xx becomes an error carrying the master's
// {"error": ...} message when present.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error != "" {
			return fmt.Errorf("%s %s: %s", method, path, e.Error)
		}
		return fmt.Errorf("%s %s: %s", method, path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
