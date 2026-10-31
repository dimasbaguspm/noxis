// Package splunk runs SPL searches against a Splunk deployment and reviews a delete. The dry run is the impact count the
// same search reports without the delete command, because a search platform has no rollback: asking how many events
// match is the only honest proof that a delete will remove what the reviewer thinks.
package splunk

import (
	"context"
	"crypto/tls"
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

	// maxHits bounds how many events a read renders.
	maxHits = 200

	// countCommand asks the same search how many events it matches: the impact a delete would have.
	countCommand = " | stats count"
	// deleteCommand removes what the search matches, which needs the can_delete capability on the account.
	deleteCommand = " | delete"
)

// DB talks to a Splunk management API.
type DB struct{}

// New builds the Splunk module.
func New() *DB { return &DB{} }

var (
	_ interfaces.Module       = (*DB)(nil)
	_ interfaces.SearchEngine = (*DB)(nil)
)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverSplunk }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool { return strings.EqualFold(driver, constants.DriverSplunk) }

// IsWrite reports whether an op mutates the deployment; a read is refused on the change path.
func (db *DB) IsWrite(op string) bool { return op == constants.SearchOpDelete }

// Ping verifies the management API answers, which is also how the version is discovered.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := call(ctx, c, http.MethodGet, "/services/server/info?output_mode=json", nil, nil); err != nil {
		return fmt.Errorf("ping %s: %w", c.Name, err)
	}
	return nil
}

// Targets lists the indexes the deployment holds: an SPL search names one to narrow what it reads.
func (db *DB) Targets(ctx context.Context, c entity.Connection) (out []entity.SearchTarget, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	var answer struct {
		Entry []struct {
			Name    string `json:"name"`
			Content struct {
				TotalEventCount string `json:"totalEventCount"`
				CurrentDBSizeMB string `json:"currentDBSizeMB"`
			} `json:"content"`
		} `json:"entry"`
	}
	if err := call(ctx, c, http.MethodGet, "/services/data/indexes?output_mode=json&count=0", nil, &answer); err != nil {
		return nil, err
	}
	out = make([]entity.SearchTarget, 0, len(answer.Entry))
	for _, entry := range answer.Entry {
		out = append(out, entity.SearchTarget{
			Name:      entry.Name,
			Documents: atoi(entry.Content.TotalEventCount),
			SizeBytes: atoi(entry.Content.CurrentDBSizeMB) * 1024 * 1024,
		})
	}
	return out, nil
}

// Search runs one SPL search and returns its events.
func (db *DB) Search(ctx context.Context, c entity.Connection, req entity.SearchRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if db.IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q changes the deployment: submit it as a request for review", req.Op)
	}
	if req.Op != constants.SearchOpSearch {
		return entity.Result{}, fmt.Errorf("%w: op %q is not a read", constants.ErrUnsupported, req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	events, err := run(ctx, c, req.Query, req.Limit)
	if err != nil {
		return entity.Result{}, err
	}
	res.Documents, res.RowsReturned = events, len(events)
	res.Message = fmt.Sprintf("%d event(s)", len(events))
	return res, nil
}

// ValidateChange runs the search without the delete command and reports how many events it matches, refusing a search
// that matches nothing: approving an empty delete is approving nothing.
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
	events, err := run(ctx, c, req.Query+countCommand, 1)
	if err != nil {
		return entity.Result{}, err
	}
	count := int64(0)
	if len(events) > 0 {
		var row struct {
			Count string `json:"count"`
		}
		if err := json.Unmarshal([]byte(events[0]), &row); err == nil {
			count = atoi(row.Count)
		}
	}
	if count == 0 {
		return entity.Result{}, fmt.Errorf("the search matches no event: nothing to review")
	}
	res.Affected = count
	res.Message = fmt.Sprintf("would delete %d event(s)", count)
	return res, nil
}

// Apply runs the search with the delete command, for real.
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
	// The delete reports its own count in the job's messages, so the impact is re-read the same way the dry run read it
	// and the change records what actually happened rather than what was asked for.
	if _, err := run(ctx, c, req.Query+deleteCommand, 1); err != nil {
		return entity.Result{}, err
	}
	events, err := run(ctx, c, req.Query+countCommand, 1)
	if err != nil {
		return entity.Result{}, err
	}
	count := int64(0)
	if len(events) > 0 {
		var row struct {
			Count string `json:"count"`
		}
		if err := json.Unmarshal([]byte(events[0]), &row); err == nil {
			count = atoi(row.Count)
		}
	}
	res.Message = fmt.Sprintf("deleted what the search matched; %d event(s) remain", count)
	return res, nil
}

// run creates a blocking search job and reads its events, so a caller never sees a job id where it asked for a result.
func run(ctx context.Context, c entity.Connection, spl string, limit int64) ([]string, error) {
	spl = strings.TrimSpace(spl)
	if spl == "" {
		return nil, fmt.Errorf("a search is required")
	}
	if !strings.HasPrefix(spl, "search ") && !strings.HasPrefix(spl, "|") {
		spl = "search " + spl
	}
	if limit <= 0 || limit > maxHits {
		limit = maxHits
	}
	form := url.Values{}
	form.Set("search", spl)
	form.Set("exec_mode", "blocking")
	form.Set("max_count", strconv.FormatInt(limit, 10))
	var created struct {
		SID string `json:"sid"`
	}
	if err := call(ctx, c, http.MethodPost, "/services/search/jobs?output_mode=json", strings.NewReader(form.Encode()), &created); err != nil {
		return nil, err
	}
	if created.SID == "" {
		return nil, fmt.Errorf("the deployment accepted no search job")
	}
	var results struct {
		Results []json.RawMessage `json:"results"`
	}
	path := "/services/search/jobs/" + url.PathEscape(created.SID) + "/results?output_mode=json&count=" + strconv.FormatInt(limit, 10)
	if err := call(ctx, c, http.MethodGet, path, nil, &results); err != nil {
		return nil, err
	}
	events := make([]string, 0, len(results.Results))
	for _, event := range results.Results {
		events = append(events, string(event))
	}
	return events, nil
}

// validateRequest checks the fields every request needs before anything is sent.
func validateRequest(req entity.SearchRequest) error {
	if strings.TrimSpace(req.Query) == "" {
		return fmt.Errorf("a search is required")
	}
	return nil
}

// atoi reads a count the deployment reports as a string, treating an absent one as zero.
func atoi(raw string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// baseURL is the management API root this connection talks to.
func baseURL(c entity.Connection) string {
	scheme := "https"
	if strings.EqualFold(c.Params["scheme"], "http") {
		scheme = "http"
	}
	return scheme + "://" + net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// client is the HTTP client this connection needs: a homelab deployment usually serves a self-signed certificate, so the
// connection can say so instead of every request failing verification.
func client(c entity.Connection) *http.Client {
	if !strings.EqualFold(c.Params["insecure"], "true") {
		return http.DefaultClient
	}
	return &http.Client{Transport: &http.Transport{
		//nolint:gosec // the operator asked for it on their own deployment
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
}

// call performs one request and decodes its JSON answer. The connection's password is the bearer token, because that is
// the field the app encrypts at rest.
func call(ctx context.Context, c entity.Connection, method, path string, body io.Reader, out any) error {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, baseURL(c)+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.Password != "" {
		req.Header.Set("Authorization", "Splunk "+c.Password)
	}
	resp, err := client(c).Do(req)
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
