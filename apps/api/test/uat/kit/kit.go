// Package kit is the acceptance suite's HTTP client: one Send for every request, headers and cookies set per test, a default bearer token that a test can override or clear, and the reply captured whole. It does not import the noxis module, so it is unit-tested against httptest without docker.
package kit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Request is one call against the API under test.
type Request struct {
	Method  string
	Path    string
	Body    []byte
	Headers map[string]string
	Cookies []*http.Cookie
	// Sign, when set, decorates the outgoing request last — after the headers, so it wins.
	Sign func(req *http.Request, body []byte)
}

// Response is the captured reply.
type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
}

// String is the trimmed body, for assertions and error messages.
func (r Response) String() string { return strings.TrimSpace(string(r.Body)) }

// Decode unmarshals a non-empty JSON body into v; an empty body is a no-op.
func (r Response) Decode(v any) error {
	if len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

// Client sends requests to one base URL. Token, when set, is sent as the default bearer; a request header overrides it, and an empty-string header removes it.
type Client struct {
	BaseURL string
	Token   string
	Headers map[string]string
	Jar     http.CookieJar
	HTTP    *http.Client
	Stub    *Stub
}

// Send performs r, or serves it from the stub store when one is attached.
func (c *Client) Send(r Request) (Response, error) {
	if c.Stub != nil {
		if !c.Stub.Record {
			return c.Stub.replay(r)
		}
		resp, err := c.doSend(r)
		if err != nil {
			return Response{}, err
		}
		if err := c.Stub.record(r, resp); err != nil {
			return Response{}, err
		}
		return resp, nil
	}
	return c.doSend(r)
}

// doSend performs r and captures the response. A JSON content type is set for a body unless a header overrides it.
func (c *Client) doSend(r Request) (Response, error) {
	var body io.Reader
	if r.Body != nil {
		body = bytes.NewReader(r.Body)
	}
	req, err := http.NewRequest(r.Method, c.BaseURL+r.Path, body)
	if err != nil {
		return Response{}, err
	}
	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	for k, v := range r.Headers {
		if v == "" {
			req.Header.Del(k)
			continue
		}
		req.Header.Set(k, v)
	}
	for _, ck := range r.Cookies {
		req.AddCookie(ck)
	}
	if r.Sign != nil {
		r.Sign(req, r.Body)
	}

	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Jar: c.Jar}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{Status: resp.StatusCode, Headers: resp.Header, Body: b}, nil
}
