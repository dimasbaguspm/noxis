package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"noxis/internal/constants"
)

// Target is one governed server as the API describes it. There is no password field: the
// API answers `has_password` and never the sealed value.
type Target struct {
	Ref             string            `json:"ref"`
	Name            string            `json:"name"`
	Driver          string            `json:"driver"`
	Host            string            `json:"host"`
	Port            int               `json:"port"`
	Database        string            `json:"database"`
	Username        string            `json:"username"`
	TLSMode         string            `json:"tls_mode"`
	Options         map[string]string `json:"options"`
	HasPassword     bool              `json:"has_password"`
	CreatedBy       string            `json:"created_by"`
	ServerVersion   string            `json:"server_version,omitempty"`
	LastProbeAt     *time.Time        `json:"last_probe_at,omitempty"`
	LastProbeStatus string            `json:"last_probe_status,omitempty"`
	ArchivedAt      *time.Time        `json:"archived_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// CreateTargetInput is a registration. Password is plaintext in transit and over TLS only.
type CreateTargetInput struct {
	Name     string            `json:"name"`
	Driver   string            `json:"driver"`
	Host     string            `json:"host"`
	Port     int               `json:"port,omitempty"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password,omitempty"`
	TLSMode  string            `json:"tls_mode,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

// ListTargetsInput is one page request: a limit, a cursor and the filters the API accepts.
type ListTargetsInput struct {
	Limit           int
	Cursor          string
	Driver          string
	Query           string
	IncludeArchived bool
}

// ListTargets answers one page of targets, newest first.
func (c *Client) ListTargets(ctx context.Context, in ListTargetsInput) (Page[Target], error) {
	query := url.Values{}
	if in.Limit > 0 {
		query.Set("limit", strconv.Itoa(in.Limit))
	}
	if in.Cursor != "" {
		query.Set("cursor", in.Cursor)
	}
	if in.Driver != "" {
		query.Set("driver", in.Driver)
	}
	if in.Query != "" {
		query.Set("q", in.Query)
	}
	if in.IncludeArchived {
		query.Set("include_archived", "true")
	}
	path := constants.PathTargets
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var items []Target
	next, err := c.do(ctx, http.MethodGet, path, nil, &items)
	return Page[Target]{Items: items, NextCursor: next}, err
}

// CreateTarget registers a target. It needs a token holding `admin`.
func (c *Client) CreateTarget(ctx context.Context, in CreateTargetInput) (Target, error) {
	var out Target
	_, err := c.do(ctx, http.MethodPost, constants.PathTargets, in, &out)
	return out, err
}

// GetTarget reads one target by reference.
func (c *Client) GetTarget(ctx context.Context, ref string) (Target, error) {
	var out Target
	_, err := c.do(ctx, http.MethodGet, targetPath(ref), nil, &out)
	return out, err
}

// UpdateTargetInput carries only the fields a caller wants to change.
type UpdateTargetInput struct {
	Name     *string            `json:"name,omitempty"`
	Username *string            `json:"username,omitempty"`
	Password *string            `json:"password,omitempty"`
	TLSMode  *string            `json:"tls_mode,omitempty"`
	Options  *map[string]string `json:"options,omitempty"`
}

// UpdateTarget renames, rotates or retunes a target.
func (c *Client) UpdateTarget(ctx context.Context, ref string, in UpdateTargetInput) (Target, error) {
	var out Target
	_, err := c.do(ctx, http.MethodPatch, targetPath(ref), in, &out)
	return out, err
}

// ArchiveTarget puts a target away. It is not deleted.
func (c *Client) ArchiveTarget(ctx context.Context, ref string) (Target, error) {
	var out Target
	_, err := c.do(ctx, http.MethodDelete, targetPath(ref), nil, &out)
	return out, err
}

func targetPath(ref string) string {
	return fmt.Sprintf("%s/%s", constants.PathTargets, url.PathEscape(ref))
}

// Capabilities is what the API says this build may offer for one target's engine. It is the
// same shape the rehearsal will consult, so nothing is offered that the engine cannot do.
type Capabilities struct {
	Driver             string   `json:"driver"`
	Flavor             string   `json:"flavor"`
	Version            string   `json:"version"`
	Major              int      `json:"major"`
	Minor              int      `json:"minor"`
	Patch              int      `json:"patch"`
	TransactionalDML   bool     `json:"transactional_dml"`
	ShadowDDLRehearsal bool     `json:"shadow_ddl_rehearsal"`
	RowImageRehearsal  bool     `json:"row_image_rehearsal"`
	IndexMetadata      bool     `json:"index_metadata"`
	ForeignKeyMetadata bool     `json:"foreign_key_metadata"`
	ExplainJSON        bool     `json:"explain_json"`
	Notes              []string `json:"notes"`
}

// TargetCapabilities answers what the API may offer for the target's engine, without opening
// a connection.
func (c *Client) TargetCapabilities(ctx context.Context, ref string) (Capabilities, error) {
	var out Capabilities
	_, err := c.do(ctx, http.MethodGet, targetPath(ref)+"/capabilities", nil, &out)
	return out, err
}

// ProbeResult is what a probe measured and the target as it now stands.
type ProbeResult struct {
	Target       Target       `json:"target"`
	Capabilities Capabilities `json:"capabilities"`
}

// ProbeTarget asks the API to connect once and record what the server says it is. A refusal is
// a 422 whose message comes from the driver, and the target records the failed attempt.
func (c *Client) ProbeTarget(ctx context.Context, ref string) (ProbeResult, error) {
	var out ProbeResult
	_, err := c.do(ctx, http.MethodPost, targetPath(ref)+"/probe", nil, &out)
	return out, err
}

// Rows is one bounded result set. Truncated is the honest answer where a keyset cannot be
// inferred from arbitrary SQL.
type Rows struct {
	Columns    []Column    `json:"columns"`
	Values     [][]*string `json:"values"`
	RowCount   int         `json:"row_count"`
	Truncated  bool        `json:"truncated"`
	DurationMS int64       `json:"duration_ms"`
}

// Column is one result column, typed as the engine names it.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Table is one table in a schema.
type Table struct {
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"size_bytes"`
}

// ColumnInfo is a column's definition.
type ColumnInfo struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Key      string  `json:"key"`
	Extra    string  `json:"extra"`
	Default  *string `json:"default"`
}

// IndexInfo is one index, in key order.
type IndexInfo struct {
	Name    string   `json:"name"`
	Unique  bool     `json:"unique"`
	Columns []string `json:"columns"`
}

// TableDetail is a table's columns, indexes and DDL.
type TableDetail struct {
	Table
	Columns   []ColumnInfo `json:"columns"`
	Indexes   []IndexInfo  `json:"indexes"`
	CreateSQL string       `json:"create_sql"`
}

// Query runs one statement the server will refuse unless it is a read.
func (c *Client) Query(ctx context.Context, ref, sql string, limit int) (Rows, error) {
	var out Rows
	body := struct {
		SQL   string `json:"sql"`
		Limit int    `json:"limit,omitempty"`
	}{SQL: sql, Limit: limit}
	_, err := c.do(ctx, http.MethodPost, targetPath(ref)+"/query", body, &out)
	return out, err
}

// Databases lists the schemas behind one target.
func (c *Client) Databases(ctx context.Context, ref string) ([]string, error) {
	var out []string
	_, err := c.do(ctx, http.MethodGet, targetPath(ref)+"/databases", nil, &out)
	return out, err
}

// Tables lists the tables in one schema.
func (c *Client) Tables(ctx context.Context, ref, database string) ([]Table, error) {
	var out []Table
	_, err := c.do(ctx, http.MethodGet, targetPath(ref)+"/databases/"+url.PathEscape(database)+"/tables", nil, &out)
	return out, err
}

// Table describes one table.
func (c *Client) Table(ctx context.Context, ref, database, table string) (TableDetail, error) {
	var out TableDetail
	_, err := c.do(ctx, http.MethodGet,
		targetPath(ref)+"/databases/"+url.PathEscape(database)+"/tables/"+url.PathEscape(table), nil, &out)
	return out, err
}
