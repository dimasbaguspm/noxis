// Package couchdb explores databases and documents over CouchDB's HTTP API, and proves a document write in a throwaway
// database before it is applied. shortcut: the console's document seam is shaped like MongoDB's, so the one keyspace a
// database holds answers the collection slot instead of inventing a hierarchy CouchDB does not have.
package couchdb

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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

	// documentsTarget names what the module actually reads: a CouchDB database is one keyspace of documents, so this is
	// the name the console's collection slot receives rather than a made-up hierarchy.
	documentsTarget = "_all_docs"

	// maxGridRows bounds how many documents a read renders.
	maxGridRows = 200

	// scratchPrefix names the throwaway database each write is proved in.
	scratchPrefix = "noxis_verify_"
)

// DB talks to CouchDB servers.
type DB struct{}

// New builds the CouchDB module.
func New() *DB { return &DB{} }

var (
	_ interfaces.Module            = (*DB)(nil)
	_ interfaces.DocumentBrowser   = (*DB)(nil)
	_ interfaces.MongoChangeEngine = (*DB)(nil)
)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverCouchDB }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool { return strings.EqualFold(driver, constants.DriverCouchDB) }

// IsWrite reports whether an op mutates documents; a read is refused on the change path.
func (db *DB) IsWrite(op string) bool { return IsWrite(op) }

// IsWrite reports whether an op mutates documents.
func IsWrite(op string) bool {
	switch op {
	case constants.MongoInsert, constants.MongoUpdate, constants.MongoDelete:
		return true
	}
	return false
}

// Ping verifies a connection answers with its welcome document.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	var welcome struct {
		CouchDB string `json:"couchdb"`
	}
	if err := call(ctx, c, http.MethodGet, "/", nil, &welcome); err != nil {
		return fmt.Errorf("ping %s: %w", c.Name, err)
	}
	return nil
}

// databaseInfo is the part of a database document the console shows.
type databaseInfo struct {
	DocCount int64 `json:"doc_count"`
	Sizes    struct {
		File     int64 `json:"file"`
		Active   int64 `json:"active"`
		External int64 `json:"external"`
	} `json:"sizes"`
}

// Databases lists the databases the server holds, with the count and size the server reports for each.
func (db *DB) Databases(ctx context.Context, c entity.Connection) (out []entity.Database, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	var names []string
	if err := call(ctx, c, http.MethodGet, "/_all_dbs", nil, &names); err != nil {
		return nil, err
	}
	out = make([]entity.Database, 0, len(names))
	for _, name := range names {
		info, err := db.info(ctx, c, name)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.Database{
			Name:      name,
			SizeBytes: info.Sizes.File,
			Objects:   info.DocCount,
		})
	}
	return out, nil
}

// Collections answers the console's collection slot with the one keyspace a CouchDB database holds.
func (db *DB) Collections(ctx context.Context, c entity.Connection, database string) (out []entity.Collection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	info, err := db.info(ctx, c, database)
	if err != nil {
		return nil, err
	}
	return []entity.Collection{{
		Name:      documentsTarget,
		Type:      "documents",
		Count:     info.DocCount,
		SizeBytes: info.Sizes.File,
	}}, nil
}

// info reads one database's own document, which is where CouchDB reports its count and size.
func (db *DB) info(ctx context.Context, c entity.Connection, database string) (databaseInfo, error) {
	var info databaseInfo
	if err := call(ctx, c, http.MethodGet, "/"+url.PathEscape(database), nil, &info); err != nil {
		return databaseInfo{}, err
	}
	return info, nil
}

// Run executes a read; a write is refused here by design.
func (db *DB) Run(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q changes documents: submit it as a request for review", req.Op)
	}
	return db.run(ctx, c, req)
}

// run performs one read against a database.
func (db *DB) run(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	// CouchDB has no count or aggregate endpoint: a filtered count is a find, and pretending otherwise would invent a
	// number this module cannot check.
	if req.Op != constants.MongoFind {
		return entity.Result{}, fmt.Errorf("%w: CouchDB serves %s, not %s", constants.ErrUnsupported, constants.MongoFind, req.Op)
	}
	docs, err := find(ctx, c, req.Database, req)
	if err != nil {
		return entity.Result{}, err
	}
	res.Documents, res.RowsReturned = docs, len(docs)
	res.Message = fmt.Sprintf("%d document(s)", len(docs))
	return res, nil
}

// find runs a Mango selector and returns the documents as JSON.
func find(ctx context.Context, c entity.Connection, database string, req entity.MongoRequest) ([]string, error) {
	body := map[string]any{}
	if selector, err := object(req.Filter); err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	} else if selector != nil {
		body["selector"] = selector
	}
	if sort, err := object(req.Sort); err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	} else if sort != nil {
		body["sort"] = sort
	}
	limit := req.Limit
	if limit <= 0 || limit > maxGridRows {
		limit = maxGridRows
	}
	body["limit"] = limit
	var answer struct {
		Docs []json.RawMessage `json:"docs"`
	}
	if err := call(ctx, c, http.MethodPost, "/"+url.PathEscape(database)+"/_find", body, &answer); err != nil {
		return nil, err
	}
	docs := make([]string, 0, len(answer.Docs))
	for _, doc := range answer.Docs {
		docs = append(docs, string(doc))
	}
	return docs, nil
}

// ValidateRequest creates a throwaway database, copies the document the write targets into it, applies the write there,
// then drops it. A CouchDB write has no rollback, so a disposable copy is the only honest proof.
func (db *DB) ValidateRequest(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	if !IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q reads only: run it in Explore", req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()

	scratch := scratchPrefix + hexSuffix()
	if err := call(ctx, c, http.MethodPut, "/"+url.PathEscape(scratch), nil, nil); err != nil {
		return entity.Result{}, fmt.Errorf("create scratch database: %w", err)
	}
	// Cleanup must survive a cancelled request: the database exists on the server the moment it is created.
	defer func() {
		_ = call(context.WithoutCancel(ctx), c, http.MethodDelete, "/"+url.PathEscape(scratch), nil, nil)
	}()

	if err := mirror(ctx, c, req, scratch); err != nil {
		return entity.Result{}, err
	}
	affected, err := write(ctx, c, scratch, req)
	if err != nil {
		return entity.Result{}, err
	}
	res.Affected = affected
	res.ScratchSchema = scratch
	res.Message = fmt.Sprintf("would affect %d document(s), checked in scratch database %q", affected, scratch)
	return res, nil
}

// Apply runs a write for real, after review.
func (db *DB) Apply(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	if !IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q reads only: run it in Explore", req.Op)
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()
	affected, err := write(ctx, c, req.Database, req)
	if err != nil {
		return entity.Result{}, err
	}
	res.Affected = affected
	res.Message = fmt.Sprintf("%d document(s) %s", affected, req.Op)
	return res, nil
}

// mirror copies the document an update or a delete targets into the scratch database, so the proof runs against the
// same revision the real write would see. An insert has nothing to mirror.
func mirror(ctx context.Context, c entity.Connection, req entity.MongoRequest, scratch string) error {
	if req.Op == constants.MongoInsert {
		return nil
	}
	id, err := documentID(req)
	if err != nil {
		return err
	}
	doc, err := read(ctx, c, req.Database, id)
	if err != nil {
		return fmt.Errorf("read %s/%s to mirror: %w", req.Database, id, err)
	}
	return call(ctx, c, http.MethodPut, "/"+url.PathEscape(scratch)+"/"+url.PathEscape(id), doc, nil)
}

// write applies one write request against a database and returns how many documents it touched.
func write(ctx context.Context, c entity.Connection, database string, req entity.MongoRequest) (int64, error) {
	value, err := object(req.Value)
	if err != nil {
		return 0, fmt.Errorf("value: %w", err)
	}
	switch req.Op {
	case constants.MongoInsert:
		if value == nil {
			return 0, fmt.Errorf("insert needs a document body")
		}
		if err := call(ctx, c, http.MethodPost, "/"+url.PathEscape(database), value, nil); err != nil {
			return 0, err
		}
		return 1, nil
	case constants.MongoUpdate:
		if value == nil {
			return 0, fmt.Errorf("update needs a document body")
		}
		id, err := documentID(req)
		if err != nil {
			return 0, err
		}
		current, err := read(ctx, c, database, id)
		if err != nil {
			return 0, fmt.Errorf("read %s/%s to update: %w", database, id, err)
		}
		// An update merges the body over the stored document: CouchDB replaces a whole document, so a field left out of
		// the body would be dropped, and a reviewer approving a patch does not expect that.
		for key, v := range value {
			current[key] = v
		}
		if err := call(ctx, c, http.MethodPut, "/"+url.PathEscape(database)+"/"+url.PathEscape(id), current, nil); err != nil {
			return 0, err
		}
		return 1, nil
	case constants.MongoDelete:
		id, err := documentID(req)
		if err != nil {
			return 0, err
		}
		current, err := read(ctx, c, database, id)
		if err != nil {
			return 0, fmt.Errorf("read %s/%s to delete: %w", database, id, err)
		}
		rev, _ := current["_rev"].(string)
		if rev == "" {
			return 0, fmt.Errorf("document %s carries no revision", id)
		}
		if err := call(ctx, c, http.MethodDelete, "/"+url.PathEscape(database)+"/"+url.PathEscape(id)+"?rev="+url.QueryEscape(rev), nil, nil); err != nil {
			return 0, err
		}
		return 1, nil
	}
	return 0, fmt.Errorf("%w: op %q is not a write", constants.ErrUnsupported, req.Op)
}

// read fetches one document by id.
func read(ctx context.Context, c entity.Connection, database, id string) (map[string]any, error) {
	var doc map[string]any
	if err := call(ctx, c, http.MethodGet, "/"+url.PathEscape(database)+"/"+url.PathEscape(id), nil, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// documentID resolves which document a write targets: the body's _id for an insert or an update, the filter's _id for a
// delete.
func documentID(req entity.MongoRequest) (string, error) {
	for _, raw := range []string{req.Value, req.Filter} {
		body, err := object(raw)
		if err != nil {
			return "", err
		}
		if id, ok := body["_id"].(string); ok && id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("the request names no document: put _id in the %s", "value or filter")
}

// validateRequest checks the fields every request needs before anything is sent.
func validateRequest(req entity.MongoRequest) error {
	if strings.TrimSpace(req.Database) == "" {
		return fmt.Errorf("a database is required")
	}
	if req.Many {
		return fmt.Errorf("%w: CouchDB writes one document per request", constants.ErrUnsupported)
	}
	return nil
}

// object decodes a JSON object, treating an empty string as absent.
func object(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// hexSuffix names one scratch database: short enough to read, random enough that two submissions never collide.
func hexSuffix() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf[:])
}

// baseURL is the server root this connection talks to.
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
