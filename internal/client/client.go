// Package client is the typed REST client both the CLI and the acceptance steps use. It
// knows the envelope, the bearer header, the cursor and the error codes, and nothing else:
// no database access, no state file, no retry policy nobody asked for.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"noxis/internal/common/ref"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"strings"
	"time"
)

// Error is a refusal from the API, carrying the status and the code it answered with so a
// caller can branch on the code instead of parsing a message.
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

// Page is one cursor page and the cursor that follows it. NextCursor is empty on the last
// page: there is no offset anywhere in this client because there is none on the wire.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// pageMeta is the envelope's meta, and it is absent rather than empty on the last page: the API says
// "no cursor" by not saying it at all.
type pageMeta struct {
	NextCursor string `json:"next_cursor"`
}

type pageEnvelope[T any] struct {
	Data []T       `json:"data"`
	Meta *pageMeta `json:"meta,omitempty"`
}

// MarshalJSON answers the shape the API sent, so `--json` on any list is the same document an agent
// would have read from the wire. Without this, the CLI printed Go field names (`Items`, `NextCursor`)
// and every reader needed a second parser for the same data.
func (p Page[T]) MarshalJSON() ([]byte, error) {
	out := pageEnvelope[T]{Data: p.Items}
	if p.NextCursor != "" {
		out.Meta = &pageMeta{NextCursor: p.NextCursor}
	}
	return json.Marshal(out)
}

// Client talks to one noxisd.
type Client struct {
	base  *url.URL
	token string
	http  *http.Client
	// lastStatus is the status of the most recent response, kept for callers that need to
	// tell a 201 from a 200.
	lastStatus int
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

// LastStatus is the status the API answered with last, or zero before any call.
func (c *Client) LastStatus() int { return c.lastStatus }

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
	_, err := c.do(ctx, http.MethodGet, pathWhoami, nil, &out)
	return out, err
}

// Live asks the public health route whether the process is up.
func (c *Client) Live(ctx context.Context) (Health, error) {
	var out Health
	_, err := c.do(ctx, http.MethodGet, pathHealthz, nil, &out)
	return out, err
}

// OpenAPI answers the document the server serves: the contract that deployment implements, not a copy
// this binary happened to be built with.
func (c *Client) OpenAPI(ctx context.Context) ([]byte, error) {
	raw, _, err := c.send(ctx, http.MethodGet, pathOpenAPI, nil, "")
	return raw, err
}

// do performs one call and returns the cursor of the page it answered, when it answered one.
func (c *Client) do(ctx context.Context, method, path string, body, out any) (string, error) {
	return c.doKey(ctx, method, path, body, out, "")
}

// doKey is do with a name for the request. A create that carries a key the server has already seen is
// answered with the row the first call made, so a retry after a timeout costs nothing.
func (c *Client) doKey(ctx context.Context, method, path string, body, out any, key string) (string, error) {
	raw, status, err := c.send(ctx, method, path, body, key)
	if err != nil {
		return "", err
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta *struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		// Not our envelope: a proxy or a bare http.Error answered instead.
		return "", &Error{Status: status, Message: snippet(raw)}
	}
	if envelope.Error != nil {
		return "", &Error{Status: status, Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	if status >= http.StatusMultipleChoices {
		return "", &Error{Status: status, Message: snippet(raw)}
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return "", fmt.Errorf("client: decode response: %w", err)
		}
	}
	if envelope.Meta != nil {
		return envelope.Meta.NextCursor, nil
	}
	return "", nil
}

// send is the one place a request is built: the bearer, the content type, the body, the name a retry
// carries, and a bounded read. It answers the body and the status, because a caller reading a document
// rather than an envelope still needs the status to tell a refusal from an answer.
func (c *Client) send(ctx context.Context, method, path string, body any, key string) (raw []byte, status int, err error) {
	ctx, end := telemetry.Span(ctx, "client.Client.Do")
	defer func() { end(err) }()

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("client: encode request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base.String()+path, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("client: request: %w", err)
	}
	req.Header.Set("Accept", constants.MediaTypeJSON)
	if body != nil {
		req.Header.Set(constants.HeaderContentType, constants.MediaTypeJSON)
	}
	if c.token != "" {
		req.Header.Set(constants.HeaderAuthorization, fmt.Sprintf("%s %s", constants.SchemeBearer, c.token))
	}
	if key != "" {
		req.Header.Set(constants.HeaderIdempotencyKey, key)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("client: %s %s: %w", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	c.lastStatus = res.StatusCode

	raw, err = io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("client: read response: %w", err)
	}
	return raw, res.StatusCode, nil
}

func snippet(raw []byte) string {
	const limit = 200
	s := strings.TrimSpace(string(raw))
	if len(s) > limit {
		return s[:limit] + "…"
	}
	return s
}

// requestKey names a create: the key the caller named, or a fresh one. A caller that sends the same key
// twice gets one row back; a caller that names none is still protected from a double-send inside one
// process, because the second call carries the key the first one made.
func requestKey(key string) string {
	if key != "" {
		return key
	}
	return ref.New()
}
