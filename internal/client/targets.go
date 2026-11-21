package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
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
	// IdempotencyKey names this registration and never travels in the body: it is the header the
	// server stores, so the same key twice is one instance.
	IdempotencyKey string `json:"-"`
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
	path := pathTargets
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var items []Target
	next, err := c.do(ctx, http.MethodGet, path, nil, &items)
	return Page[Target]{Items: items, NextCursor: next}, err
}

// CreateTarget registers a target. It needs a token holding `admin`. The key names the request, so a
// create that is retried after a timeout answers the row the first call made.
func (c *Client) CreateTarget(ctx context.Context, in CreateTargetInput) (Target, error) {
	var out Target
	_, err := c.doKey(ctx, http.MethodPost, pathTargets, in, &out, requestKey(in.IdempotencyKey))
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
	return fmt.Sprintf("%s/%s", pathTargets, url.PathEscape(ref))
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

// TestTargetInput is a connection nobody stored: a registration without the name, because a name is
// the row's business and this call writes no row.
type TestTargetInput struct {
	Driver   string            `json:"driver"`
	Host     string            `json:"host"`
	Port     int               `json:"port,omitempty"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password,omitempty"`
	TLSMode  string            `json:"tls_mode,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

// TestTargetResult is what a dialled configuration answered: the engine it reached, and the schemas
// those credentials can see. An empty list is still a successful test.
type TestTargetResult struct {
	Capabilities Capabilities `json:"capabilities"`
	Databases    []string     `json:"databases"`
}

// TestTarget dials the credentials as typed and answers what the engine said. Nothing is written: the
// console calls this before it registers an instance, and the CLI calls it for the same reason.
func (c *Client) TestTarget(ctx context.Context, in TestTargetInput) (TestTargetResult, error) {
	var out TestTargetResult
	_, err := c.do(ctx, http.MethodPost, pathTargets+"/test", in, &out)
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
	_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/databases/%s/tables", targetPath(ref), url.PathEscape(database)), nil, &out)
	return out, err
}

// Table describes one table.
func (c *Client) Table(ctx context.Context, ref, database, table string) (TableDetail, error) {
	var out TableDetail
	_, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("%s/databases/%s/tables/%s", targetPath(ref), url.PathEscape(database), url.PathEscape(table)), nil, &out)
	return out, err
}

// Proposal is one proposed change and the verdict of its last rehearsal.
type Proposal struct {
	Ref           string     `json:"ref"`
	TargetRef     string     `json:"target_ref"`
	Title         string     `json:"title"`
	Kind          string     `json:"kind"`
	Query         string     `json:"query"`
	Rollback      string     `json:"rollback"`
	Note          string     `json:"note"`
	Status        string     `json:"status"`
	Tier          string     `json:"tier"`
	Irreversible  bool       `json:"irreversible"`
	ProposedBy    string     `json:"proposed_by"`
	RehearsalRef  string     `json:"rehearsal_ref,omitempty"`
	DecidedBy     string     `json:"decided_by,omitempty"`
	DecidedAt     *time.Time `json:"decided_at,omitempty"`
	DecidedNote   string     `json:"decided_note,omitempty"`
	AppliedBy     string     `json:"applied_by,omitempty"`
	Algorithm     string     `json:"algorithm,omitempty"`
	RowsEstimated int64      `json:"rows_estimated,omitempty"`
	RowsAffected  int64      `json:"rows_affected,omitempty"`
	AppliedAt     *time.Time `json:"applied_at,omitempty"`
	Failure       string     `json:"failure,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Rehearsal     *Rehearsal `json:"rehearsal,omitempty"`
}

// Rehearsal is the evidence behind a proposal.
type Rehearsal struct {
	Ref          string    `json:"ref"`
	ProposalRef  string    `json:"proposal_ref"`
	Tier         string    `json:"tier"`
	Status       string    `json:"status"`
	SchemaDiff   string    `json:"schema_diff,omitempty"`
	RowDiff      string    `json:"row_diff,omitempty"`
	Plan         string    `json:"plan,omitempty"`
	Irreversible bool      `json:"irreversible"`
	Note         string    `json:"note"`
	DurationMS   int64     `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateProposalInput is one proposal as a caller writes it. IdempotencyKey names the request and
// never travels in the body.
type CreateProposalInput struct {
	Title          string `json:"title"`
	Query          string `json:"query"`
	Rollback       string `json:"rollback,omitempty"`
	Note           string `json:"note,omitempty"`
	Tier           string `json:"tier,omitempty"`
	IdempotencyKey string `json:"-"`
}

// CreateProposal stores and rehearses one change. The key names the request, so a retry rehearses
// nothing and answers the proposal the first call made.
func (c *Client) CreateProposal(ctx context.Context, targetRef string, in CreateProposalInput) (Proposal, error) {
	var out Proposal
	_, err := c.doKey(ctx, http.MethodPost, targetPath(targetRef)+"/proposals", in, &out,
		requestKey(in.IdempotencyKey))
	return out, err
}

// ListProposals answers one page of the queue.
func (c *Client) ListProposals(ctx context.Context, limit int, cursor, status, kind, target string, mine bool) (Page[Proposal], error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if status != "" {
		query.Set("status", status)
	}
	if kind != "" {
		query.Set("kind", kind)
	}
	if target != "" {
		query.Set("target", target)
	}
	if mine {
		query.Set("mine", "true")
	}
	path := pathProposals
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var items []Proposal
	next, err := c.do(ctx, http.MethodGet, path, nil, &items)
	return Page[Proposal]{Items: items, NextCursor: next}, err
}

// GetProposal reads one proposal with its rehearsal.
func (c *Client) GetProposal(ctx context.Context, ref string) (Proposal, error) {
	var out Proposal
	_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s", pathProposals, url.PathEscape(ref)), nil, &out)
	return out, err
}

// RehearseProposal re-runs the proof, optionally at a different tier.
func (c *Client) RehearseProposal(ctx context.Context, ref, tier string) (Proposal, error) {
	var out Proposal
	body := struct {
		Tier string `json:"tier,omitempty"`
	}{Tier: tier}
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/rehearse", pathProposals, url.PathEscape(ref)), body, &out)
	return out, err
}

// Event is one line of a proposal's append-only history.
type Event struct {
	Ref         string            `json:"ref"`
	SubjectRef  string            `json:"subject_ref"`
	SubjectKind string            `json:"subject_kind"`
	Kind        string            `json:"kind"`
	Actor       string            `json:"actor"`
	Detail      map[string]string `json:"detail"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ApproveProposal records approval. The server refuses a token approving its own proposal unless it
// is an admin, and journals that exception.
func (c *Client) ApproveProposal(ctx context.Context, ref, note string) (Proposal, error) {
	return c.decide(ctx, ref, "approve", note)
}

// RejectProposal records a refusal, which may be the proposer withdrawing.
func (c *Client) RejectProposal(ctx context.Context, ref, note string) (Proposal, error) {
	return c.decide(ctx, ref, "reject", note)
}

func (c *Client) decide(ctx context.Context, ref, decision, note string) (Proposal, error) {
	var out Proposal
	body := struct {
		Note string `json:"note,omitempty"`
	}{Note: note}
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/%s", pathProposals, url.PathEscape(ref), decision), body, &out)
	return out, err
}

// ApplyProposal runs the approved statement. confirm must be true for a change the rehearsal marked
// irreversible.
func (c *Client) ApplyProposal(ctx context.Context, ref string, confirm bool) (Proposal, error) {
	var out Proposal
	body := struct {
		ConfirmDestructive bool `json:"confirm_destructive,omitempty"`
	}{ConfirmDestructive: confirm}
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/apply", pathProposals, url.PathEscape(ref)), body, &out)
	return out, err
}

// ProposalEvents answers the history of one proposal, newest first.
func (c *Client) ProposalEvents(ctx context.Context, ref string) ([]Event, error) {
	var out []Event
	_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s/events", pathProposals, url.PathEscape(ref)), nil, &out)
	return out, err
}
