// Package elasticsearch browses the indices a cluster holds, runs one query document against an index, and reviews a
// delete by query. The dry run is the impact count the cluster itself reports, because a search platform has no
// rollback: a disposable copy of an index would prove less than asking the cluster how many documents match.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const (
	pingTimeout = 10 * time.Second
	opTimeout   = 30 * time.Second

	// maxHits bounds how many documents a read renders.
	maxHits = 200

	// matchAll is the query an empty request means: everything the target holds.
	matchAll = `{"match_all":{}}`
)

// DB talks to Elasticsearch clusters.
type DB struct{}

// New builds the Elasticsearch module.
func New() *DB { return &DB{} }

var (
	_ interfaces.Module       = (*DB)(nil)
	_ interfaces.SearchEngine = (*DB)(nil)
)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverElasticsearch }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool {
	return strings.EqualFold(driver, constants.DriverElasticsearch)
}

// IsWrite reports whether an op mutates the cluster; a read is refused on the change path.
func (db *DB) IsWrite(op string) bool { return op == constants.SearchOpDelete }

// Ping verifies a cluster answers with its own version.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	var info struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	if err := call(ctx, c, http.MethodGet, "/", nil, &info); err != nil {
		return fmt.Errorf("ping %s: %w", c.Name, err)
	}
	return nil
}

// Targets lists the indices the cluster holds, with the count and size its own catalog reports.
func (db *DB) Targets(ctx context.Context, c entity.Connection) (out []entity.SearchTarget, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	var rows []struct {
		Index string `json:"index"`
		Docs  string `json:"docs.count"`
		Size  string `json:"store.size"`
	}
	path := "/_cat/indices?format=json&bytes=b&h=index,docs.count,store.size&expand_wildcards=all"
	if err := call(ctx, c, http.MethodGet, path, nil, &rows); err != nil {
		return nil, err
	}
	out = make([]entity.SearchTarget, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(row.Index, ".") {
			continue // a dot index is the cluster's own bookkeeping, not something an operator reads
		}
		out = append(out, entity.SearchTarget{
			Name:      row.Index,
			Documents: atoi(row.Docs),
			SizeBytes: atoi(row.Size),
		})
	}
	return out, nil
}

// Search runs a read against one index.
func (db *DB) Search(ctx context.Context, c entity.Connection, req entity.SearchRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if db.IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q changes the cluster: submit it as a request for review", req.Op)
	}
	if req.Op != constants.SearchOpSearch {
		return entity.Result{}, fmt.Errorf("%w: op %q is not a read", constants.ErrUnsupported, req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	query, err := queryOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	limit := req.Limit
	if limit <= 0 || limit > maxHits {
		limit = maxHits
	}
	var answer struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	body := map[string]any{"query": query, "size": limit}
	if err := call(ctx, c, http.MethodPost, "/"+url.PathEscape(req.Target)+"/_search", body, &answer); err != nil {
		return entity.Result{}, err
	}
	res.Documents = make([]string, 0, len(answer.Hits.Hits))
	for _, hit := range answer.Hits.Hits {
		res.Documents = append(res.Documents, string(hit.Source))
	}
	res.RowsReturned, res.EstimatedRows = len(res.Documents), answer.Hits.Total.Value
	res.Message = fmt.Sprintf("%d of %d document(s)", len(res.Documents), answer.Hits.Total.Value)
	return res, nil
}

// ValidateChange asks the cluster how many documents the delete would remove, and refuses a query that matches nothing:
// an approval of an empty delete is an approval of nothing.
func (db *DB) ValidateChange(ctx context.Context, c entity.Connection, req entity.SearchRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	if !db.IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q reads only: run it in the search page", req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	query, err := queryOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	var answer struct {
		Count int64 `json:"count"`
	}
	if err := call(ctx, c, http.MethodPost, "/"+url.PathEscape(req.Target)+"/_count", map[string]any{"query": query}, &answer); err != nil {
		return entity.Result{}, err
	}
	if answer.Count == 0 {
		return entity.Result{}, fmt.Errorf("the query matches no document in %q: nothing to review", req.Target)
	}
	res.Affected = answer.Count
	res.Message = fmt.Sprintf("would delete %d document(s) from %q", answer.Count, req.Target)
	return res, nil
}

// Apply runs a reviewed delete by query for real.
func (db *DB) Apply(ctx context.Context, c entity.Connection, req entity.SearchRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	if !db.IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q reads only: run it in the search page", req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	query, err := queryOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	var answer struct {
		Deleted int64 `json:"deleted"`
	}
	path := "/" + url.PathEscape(req.Target) + "/_delete_by_query?refresh=true&conflicts=proceed"
	if err := call(ctx, c, http.MethodPost, path, map[string]any{"query": query}, &answer); err != nil {
		return entity.Result{}, err
	}
	res.Affected = answer.Deleted
	res.Message = fmt.Sprintf("%d document(s) deleted from %q", answer.Deleted, req.Target)
	return res, nil
}

// validateRequest checks the fields every request needs before anything is sent.
func validateRequest(req entity.SearchRequest) error {
	if strings.TrimSpace(req.Target) == "" {
		return fmt.Errorf("a target index is required")
	}
	return nil
}

// queryOf decodes the request's query document, so a malformed one is refused before it reaches the cluster.
func queryOf(req entity.SearchRequest) (any, error) {
	raw := strings.TrimSpace(req.Query)
	if raw == "" {
		raw = matchAll
	}
	var query any
	if err := json.Unmarshal([]byte(raw), &query); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return query, nil
}

// atoi reads a count the cluster reports as a string, treating an absent one as zero.
func atoi(raw string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// baseURL is the cluster root this connection talks to.
func baseURL(c entity.Connection) string {
	scheme := "http"
	if strings.EqualFold(c.Params["scheme"], "https") {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// call performs one request and decodes its JSON answer, so every operation shares the auth, the timeout and the error
// shape.
func call(ctx context.Context, c entity.Connection, method, path string, body, out any) error {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL(c)+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.User != "" {
		req.SetBasicAuth(c.User, c.Password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s %s: %w", method, path, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}
