// Package client is the typed REST client both the CLI and the acceptance steps use. It
// knows the envelope, the bearer header and the error codes, and nothing else: no
// database access, no state file, no retry policy nobody asked for.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"noxis/internal/constants"
)

// Error is a refusal from the API, carrying the status and the code it answered with so
// a caller can branch on the code instead of parsing a message.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("api: %d %s", e.Status, http.StatusText(e.Status))
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Client talks to one noxisd.
type Client struct {
	base  *url.URL
	token string
	http  *http.Client
}

// New builds a client. An empty token is allowed: a public route needs none.
func New(baseURL, token string) (*Client, error) {
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("client: base url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("client: %q is not an absolute url", baseURL)
	}
	return &Client{base: base, token: token, http: &http.Client{Timeout: 60 * time.Second}}, nil
}

// Identity is the caller the API resolved for the token in use.
type Identity struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// Health is the public liveness answer.
type Health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Whoami asks the API which caller this token is.
func (c *Client) Whoami(ctx context.Context) (Identity, error) {
	var out Identity
	return out, c.do(ctx, http.MethodGet, constants.PathWhoami, nil, &out)
}

// Live asks the public health route whether the process is up.
func (c *Client) Live(ctx context.Context) (Health, error) {
	var out Health
	return out, c.do(ctx, http.MethodGet, constants.PathHealthz, nil, &out)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("client: encode request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base.String()+path, reader)
	if err != nil {
		return fmt.Errorf("client: request: %w", err)
	}
	req.Header.Set("Accept", constants.MediaTypeJSON)
	if body != nil {
		req.Header.Set(constants.HeaderContentType, constants.MediaTypeJSON)
	}
	if c.token != "" {
		req.Header.Set(constants.HeaderAuthorization, constants.SchemeBearer+" "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("client: %s %s: %w", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("client: read response: %w", err)
	}

	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		// Not our envelope: a proxy or a bare http.Error answered instead.
		return &Error{Status: res.StatusCode, Message: snippet(raw)}
	}
	if envelope.Error != nil {
		return &Error{Status: res.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	if res.StatusCode >= http.StatusMultipleChoices {
		return &Error{Status: res.StatusCode, Message: snippet(raw)}
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("client: decode response: %w", err)
		}
	}
	return nil
}

func snippet(raw []byte) string {
	const limit = 200
	s := strings.TrimSpace(string(raw))
	if len(s) > limit {
		return s[:limit] + "…"
	}
	return s
}
