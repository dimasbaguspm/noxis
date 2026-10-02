// Package master is the typed client for the master REST API: one method per
// endpoint instead of hand-built URLs. Transport lives in internal/common/http
// (standard headers on every request).
package master

import (
	"context"

	"noxis/internal/common/http"
	"noxis/internal/entity"
)

type Client struct{ hc *http.Client }

// NewClient returns a client for the master at base (NOXIS_MASTER).
func NewClient(base string) *Client {
	return &Client{hc: http.NewClient(base)}
}

func (c *Client) RegisterNode(ctx context.Context, node string) error {
	return c.hc.Do(ctx, "POST", "/v1/nodes/"+node+"/register", nil, nil)
}

func (c *Client) GetSpec(ctx context.Context, node string) (*entity.Spec, error) {
	var spec entity.Spec
	if err := c.hc.Do(ctx, "GET", "/v1/nodes/"+node+"/spec", nil, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (c *Client) SetSpec(ctx context.Context, node string, engines []entity.Engine) error {
	return c.hc.Do(ctx, "PUT", "/v1/nodes/"+node+"/spec", map[string]any{"engines": engines}, nil)
}

// Query runs SQL against an engine on the node.
func (c *Client) Query(ctx context.Context, node, engine, sql string) (string, error) {
	var out struct {
		Output string `json:"output"`
	}
	if err := c.hc.Do(ctx, "POST", "/v1/nodes/"+node+"/doorman/query",
		map[string]any{"engine": engine, "sql": sql}, &out); err != nil {
		return "", err
	}
	return out.Output, nil
}
