// Package meilisearch browses the indexes a server holds, runs one search against an index, and reviews a delete by
// filter. The dry run is the impact count the server estimates, because a search platform has no rollback.
package meilisearch

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

	// taskPoll and taskBudget bound how long a reviewed write waits for the server to finish the task it accepted.
	taskPoll   = 500 * time.Millisecond
	taskBudget = 30 * time.Second
)

// DB talks to Meilisearch servers.
type DB struct{}

// New builds the Meilisearch module.
func New() *DB { return &DB{} }

var (
	_ interfaces.Module       = (*DB)(nil)
	_ interfaces.SearchEngine = (*DB)(nil)
)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverMeilisearch }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool {
	return strings.EqualFold(driver, constants.DriverMeilisearch)
}

// IsWrite reports whether an op mutates the server; a read is refused on the change path.
func (db *DB) IsWrite(op string) bool { return op == constants.SearchOpDelete }

// Ping verifies a server answers its health endpoint.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := call(ctx, c, http.MethodGet, "/health", nil, nil); err != nil {
		return fmt.Errorf("ping %s: %w", c.Name, err)
	}
	return nil
}

// Targets lists the indexes the server holds, with the count its own stats report.
func (db *DB) Targets(ctx context.Context, c entity.Connection) (out []entity.SearchTarget, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	var list struct {
		Results []struct {
			UID string `json:"uid"`
		} `json:"results"`
	}
	if err := call(ctx, c, http.MethodGet, "/indexes?limit=1000", nil, &list); err != nil {
		return nil, err
	}
	out = make([]entity.SearchTarget, 0, len(list.Results))
	for _, index := range list.Results {
		var stats struct {
			Documents int64 `json:"numberOfDocuments"`
			Size      int64 `json:"rawDocumentDbSize"`
		}
		if err := call(ctx, c, http.MethodGet, "/indexes/"+url.PathEscape(index.UID)+"/stats", nil, &stats); err != nil {
			return nil, err
		}
		out = append(out, entity.SearchTarget{Name: index.UID, Documents: stats.Documents, SizeBytes: stats.Size})
	}
	return out, nil
}

// Search runs a read against one index.
func (db *DB) Search(ctx context.Context, c entity.Connection, req entity.SearchRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if db.IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q changes the server: submit it as a request for review", req.Op)
	}
	if req.Op != constants.SearchOpSearch {
		return entity.Result{}, fmt.Errorf("%w: op %q is not a read", constants.ErrUnsupported, req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	params, err := paramsOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	limit := req.Limit
	if limit <= 0 || limit > maxHits {
		limit = maxHits
	}
	params["limit"] = limit
	var answer struct {
		Hits     []json.RawMessage `json:"hits"`
		Estimate int64             `json:"estimatedTotalHits"`
	}
	path := "/indexes/" + url.PathEscape(req.Target) + "/search"
	if err := call(ctx, c, http.MethodPost, path, params, &answer); err != nil {
		return entity.Result{}, err
	}
	res.Documents = make([]string, 0, len(answer.Hits))
	for _, hit := range answer.Hits {
		res.Documents = append(res.Documents, string(hit))
	}
	res.RowsReturned, res.EstimatedRows = len(res.Documents), answer.Estimate
	res.Message = fmt.Sprintf("%d of %d document(s)", len(res.Documents), answer.Estimate)
	return res, nil
}

// ValidateChange asks the server how many documents the filter matches, and refuses a filter that matches nothing.
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
	filter, err := filterOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	var answer struct {
		Estimate int64 `json:"estimatedTotalHits"`
	}
	body := map[string]any{"filter": filter, "limit": 0}
	path := "/indexes/" + url.PathEscape(req.Target) + "/search"
	if err := call(ctx, c, http.MethodPost, path, body, &answer); err != nil {
		return entity.Result{}, err
	}
	if answer.Estimate == 0 {
		return entity.Result{}, fmt.Errorf("the filter matches no document in %q: nothing to review", req.Target)
	}
	res.Affected = answer.Estimate
	res.Message = fmt.Sprintf("would delete %d document(s) from %q", answer.Estimate, req.Target)
	return res, nil
}

// Apply runs a reviewed delete by filter and waits for the task the server accepted.
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
	filter, err := filterOf(req)
	if err != nil {
		return entity.Result{}, err
	}
	var accepted struct {
		TaskUID int64 `json:"taskUid"`
	}
	path := "/indexes/" + url.PathEscape(req.Target) + "/documents/delete"
	if err := call(ctx, c, http.MethodPost, path, map[string]any{"filter": filter}, &accepted); err != nil {
		return entity.Result{}, err
	}
	deleted, err := awaitTask(ctx, c, accepted.TaskUID)
	if err != nil {
		return entity.Result{}, err
	}
	res.Affected = deleted
	res.Message = fmt.Sprintf("%d document(s) deleted from %q", deleted, req.Target)
	return res, nil
}

// awaitTask waits for the asynchronous write the server accepted, because a task id is not an outcome: the queue
// records what happened, not what was queued.
func awaitTask(ctx context.Context, c entity.Connection, uid int64) (int64, error) {
	deadline := time.Now().Add(taskBudget)
	for {
		var task struct {
			Status string `json:"status"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
			Details struct {
				Deleted int64 `json:"deletedDocuments"`
			} `json:"details"`
		}
		if err := call(ctx, c, http.MethodGet, "/tasks/"+strconv.FormatInt(uid, 10), nil, &task); err != nil {
			return 0, err
		}
		switch task.Status {
		case "succeeded":
			return task.Details.Deleted, nil
		case "failed", "canceled":
			message := "the task did not succeed"
			if task.Error != nil && task.Error.Message != "" {
				message = task.Error.Message
			}
			return 0, fmt.Errorf("task %d %s: %s", uid, task.Status, message)
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("task %d is still %q after %s", uid, task.Status, taskBudget)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(taskPoll):
		}
	}
}

// validateRequest checks the fields every request needs before anything is sent.
func validateRequest(req entity.SearchRequest) error {
	if strings.TrimSpace(req.Target) == "" {
		return fmt.Errorf("a target index is required")
	}
	return nil
}

// paramsOf decodes the request's search parameters, so a malformed document is refused before it reaches the server.
func paramsOf(req entity.SearchRequest) (map[string]any, error) {
	raw := strings.TrimSpace(req.Query)
	if raw == "" {
		return map[string]any{}, nil
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return params, nil
}

// filterOf reads the filter a delete needs: a text query cannot express what to remove, so it is refused by name.
func filterOf(req entity.SearchRequest) (string, error) {
	params, err := paramsOf(req)
	if err != nil {
		return "", err
	}
	filter, _ := params["filter"].(string)
	if strings.TrimSpace(filter) == "" {
		return "", fmt.Errorf("%w: a delete needs a filter, and a text query cannot say what to remove", constants.ErrUnsupported)
	}
	return filter, nil
}

// baseURL is the server root this connection talks to.
func baseURL(c entity.Connection) string {
	scheme := "http"
	if strings.EqualFold(c.Params["scheme"], "https") {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// call performs one request and decodes its JSON answer. The master key travels as the connection's password, because
// that is the field the app encrypts at rest.
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
	if c.Password != "" {
		req.Header.Set("Authorization", "Bearer "+c.Password)
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
