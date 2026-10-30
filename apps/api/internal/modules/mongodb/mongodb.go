// Package mongodb explores databases, collections, documents, and server stats; shortcut: writes use scratch-copy validation until transactions are available on replica sets.
package mongodb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const (
	pingTimeout = 10 * time.Second
	opTimeout   = 30 * time.Second

	// maxGridColumns bounds a rendered document grid: the union of keys across the returned documents is capped so one wide collection cannot produce a thousand-column table.
	maxGridColumns = 40

	// maxGridRows bounds how many documents a read request renders.
	maxGridRows = 200

	// collStatWorkers bounds how many collections are described at once.
	collStatWorkers = 4
)

// DB talks to MongoDB servers.
type DB struct{}

func New() *DB { return &DB{} }

// Satisfier.
var _ interfaces.MongoModule = (*DB)(nil)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverMongo }

// IsWrite reports whether an op mutates documents.
func (db *DB) IsWrite(op string) bool { return IsWrite(op) }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool { return strings.EqualFold(driver, constants.DriverMongo) }

// Ping verifies the server answers and the credentials work.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		if err := cl.Ping(ctx, nil); err != nil {
			return fmt.Errorf("ping %s: %w", c.Name, err)
		}
		return nil
	})
}

// Databases lists the databases with their sizes.
func (db *DB) Databases(ctx context.Context, c entity.Connection) (out []entity.Database, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		names, err := cl.ListDatabaseNames(ctx, bson.D{})
		if err != nil {
			return fmt.Errorf("list databases on %s: %w", c.Name, err)
		}
		out = make([]entity.Database, 0, len(names))
		for _, name := range names {
			db := entity.Database{Name: name}
			var stats bson.M
			if err := cl.Database(name).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats); err == nil {
				db.SizeBytes = number(stats["dataSize"])
				db.StorageBytes = number(stats["storageSize"])
				db.IndexSizeBytes = number(stats["indexSize"])
				db.Collections = number(stats["collections"])
				db.Objects = number(stats["objects"])
				db.AvgObjectSize = number(stats["avgObjSize"])
			}
			// a database whose stats were refused still belongs in the list
			out = append(out, db)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return nil
	})
	return out, err
}

// Collections lists a database's collections with their sizes.
func (db *DB) Collections(ctx context.Context, c entity.Connection, database string) (out []entity.Collection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if database == "" {
		return nil, fmt.Errorf("database is required")
	}
	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		cursor, err := cl.Database(database).ListCollections(ctx, bson.D{})
		if err != nil {
			return fmt.Errorf("list collections in %s: %w", database, err)
		}
		defer func() { _ = cursor.Close(ctx) }()

		var described []entity.Collection
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return fmt.Errorf("decode collection: %w", err)
			}
			collection := entity.Collection{Name: stringOf(doc["name"]), Type: stringOf(doc["type"])}
			if collection.Type == "" {
				collection.Type = "collection"
			}
			described = append(described, collection)
		}
		if err := cursor.Err(); err != nil {
			return fmt.Errorf("iterate collections: %w", err)
		}

		// collStats is one round trip per collection; bounded workers keep a wide database from serialising.
		out = describeAll(ctx, cl.Database(database), described)
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return nil
	})
	return out, err
}

// ReadOps are the ops the console runs directly; WriteOps go through the request queue instead (validated on a scratch collection, then reviewed).
var (
	ReadOps  = []string{constants.MongoFind, constants.MongoAggregate, constants.MongoCount}
	WriteOps = []string{constants.MongoInsert, constants.MongoUpdate, constants.MongoDelete}
)

// IsWrite reports whether an op changes documents.
func IsWrite(op string) bool {
	for _, write := range WriteOps {
		if op == write {
			return true
		}
	}
	return false
}

// Run executes one read request.
func (db *DB) Run(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if IsWrite(req.Op) {
		return entity.Result{}, fmt.Errorf("op %q changes documents: submit it as a request for review", req.Op)
	}
	return db.run(ctx, c, req)
}

// ValidateRequest copies matching documents to a scratch collection, applies the write, then drops it; shortcut: this is not a transaction, so use transactions on replica sets.
func (db *DB) ValidateRequest(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()

	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()

		scratch := cl.Database("noxis_validate_" + strings.ReplaceAll(randSuffix(), "-", ""))
		defer func() { _ = scratch.Drop(context.Background()) }()
		temporary := scratch.Collection("dryrun")

		affected, err := applyOn(ctx, cl.Database(req.Database).Collection(req.Collection), temporary, req)
		if err != nil {
			return err
		}
		res.Affected = affected
		res.Message = fmt.Sprintf("would affect %d document(s), checked on a scratch collection", affected)
		return nil
	})
	return res, err
}

// Apply runs a write for real, after review.
func (db *DB) Apply(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	return db.run(ctx, c, req)
}

// applyOn stages matching documents in target and applies the write there for both dry-run and real apply.
func applyOn(ctx context.Context, source, target *mongo.Collection, req entity.MongoRequest) (int64, error) {
	if req.Op == constants.MongoInsert {
		doc, err := parseValue(req.Value)
		if err != nil {
			return 0, err
		}
		if req.Many {
			docs, ok := doc.(bson.A)
			if !ok {
				return 0, fmt.Errorf("insert many needs an array of documents in the value box")
			}
			inserted, err := target.InsertMany(ctx, docs)
			if err != nil {
				return 0, fmt.Errorf("insertMany: %w", err)
			}
			return int64(len(inserted.InsertedIDs)), nil
		}
		if _, err := target.InsertOne(ctx, doc); err != nil {
			return 0, fmt.Errorf("insertOne: %w", err)
		}
		return 1, nil
	}

	// update and delete: copy what the filter matches, then write to the copy
	filter, err := parseFilter(req.Filter)
	if err != nil {
		return 0, err
	}
	cursor, err := source.Find(ctx, filter, options.Find().SetLimit(maxGridRows*25))
	if err != nil {
		return 0, fmt.Errorf("find: %w", err)
	}
	var docs bson.A
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return 0, fmt.Errorf("decode document: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := cursor.Err(); err != nil {
		return 0, fmt.Errorf("iterate: %w", err)
	}
	_ = cursor.Close(ctx)
	if len(docs) > 0 {
		if _, err := target.InsertMany(ctx, docs); err != nil {
			return 0, fmt.Errorf("stage documents: %w", err)
		}
	}

	switch req.Op {
	case constants.MongoUpdate:
		value, err := parseValue(req.Value)
		if err != nil {
			return 0, err
		}
		if req.Many {
			result, err := target.UpdateMany(ctx, bson.M{}, value)
			if err != nil {
				return 0, fmt.Errorf("updateMany: %w", err)
			}
			return result.ModifiedCount, nil
		}
		result, err := target.UpdateOne(ctx, bson.M{}, value)
		if err != nil {
			return 0, fmt.Errorf("updateOne: %w", err)
		}
		return result.ModifiedCount, nil
	case constants.MongoDelete:
		if req.Many {
			result, err := target.DeleteMany(ctx, bson.M{})
			if err != nil {
				return 0, fmt.Errorf("deleteMany: %w", err)
			}
			return result.DeletedCount, nil
		}
		result, err := target.DeleteOne(ctx, bson.M{})
		if err != nil {
			return 0, fmt.Errorf("deleteOne: %w", err)
		}
		return result.DeletedCount, nil
	}
	return 0, fmt.Errorf("unsupported op %q", req.Op)
}

// randSuffix names a scratch database uniquely, so two validations running at once cannot drop each other's work.
func randSuffix() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

// run executes a request against the real collection.
func (db *DB) run(ctx context.Context, c entity.Connection, req entity.MongoRequest) (res entity.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()

	if err := validateRequest(req); err != nil {
		return entity.Result{}, err
	}
	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		collection := cl.Database(req.Database).Collection(req.Collection)
		switch req.Op {
		case constants.MongoFind:
			return find(ctx, collection, req, &res)
		case constants.MongoAggregate:
			return aggregate(ctx, collection, req, &res)
		case constants.MongoCount:
			filter, err := parseFilter(req.Filter)
			if err != nil {
				return err
			}
			count, err := collection.CountDocuments(ctx, filter)
			if err != nil {
				return fmt.Errorf("count: %w", err)
			}
			res.Columns = []string{"count"}
			res.Rows = [][]string{{fmt.Sprint(count)}}
			res.RowsReturned = 1
			res.Message = "counted"
			return nil
		case constants.MongoInsert:
			return insert(ctx, collection, req, &res)
		case constants.MongoUpdate:
			return update(ctx, collection, req, &res)
		case constants.MongoDelete:
			return remove(ctx, collection, req, &res)
		}
		return fmt.Errorf("unknown op %q", req.Op)
	})
	return res, err
}

// validateRequest keeps the request honest: a known op, a target, and no update or delete without a filter (an empty filter would rewrite or empty the whole collection).
func validateRequest(r entity.MongoRequest) error {
	if r.Database == "" || r.Collection == "" {
		return fmt.Errorf("database and collection are required")
	}
	switch r.Op {
	case constants.MongoFind, constants.MongoAggregate, constants.MongoCount:
		return nil
	case constants.MongoInsert, constants.MongoUpdate, constants.MongoDelete:
	default:
		return fmt.Errorf("unsupported op %q (find, aggregate, count, insert, update, delete)", r.Op)
	}
	if r.Op == constants.MongoUpdate || r.Op == constants.MongoDelete {
		filter, err := parseFilter(r.Filter)
		if err != nil {
			return err
		}
		if len(filter) == 0 {
			return fmt.Errorf("%s needs a filter: an empty one would touch every document", r.Op)
		}
	}
	if r.Op == constants.MongoUpdate && strings.TrimSpace(r.Value) == "" {
		return fmt.Errorf("update needs a value document")
	}
	if r.Op == constants.MongoInsert && strings.TrimSpace(r.Value) == "" {
		return fmt.Errorf("insert needs a value document")
	}
	return nil
}

func find(ctx context.Context, collection *mongo.Collection, req entity.MongoRequest, res *entity.Result) error {
	filter, err := parseFilter(req.Filter)
	if err != nil {
		return err
	}
	opts := options.Find()
	if req.Limit > 0 && req.Limit <= maxGridRows {
		opts.SetLimit(req.Limit)
	} else {
		opts.SetLimit(maxGridRows)
	}
	if strings.TrimSpace(req.Sort) != "" {
		var sortDoc bson.D
		if err := bson.UnmarshalExtJSON([]byte(req.Sort), true, &sortDoc); err != nil {
			return fmt.Errorf("sort is not valid JSON: %w", err)
		}
		opts.SetSort(sortDoc)
	}
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}
	return renderCursor(ctx, cursor, res)
}

func aggregate(ctx context.Context, collection *mongo.Collection, req entity.MongoRequest, res *entity.Result) error {
	if strings.TrimSpace(req.Filter) == "" {
		return fmt.Errorf("aggregate needs a pipeline in the filter box, e.g. [{$match: {}}]")
	}
	var pipeline bson.A
	if err := bson.UnmarshalExtJSON([]byte(req.Filter), true, &pipeline); err != nil {
		return fmt.Errorf("pipeline is not a valid JSON array: %w", err)
	}
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}
	return renderCursor(ctx, cursor, res)
}

func insert(ctx context.Context, collection *mongo.Collection, req entity.MongoRequest, res *entity.Result) error {
	doc, err := parseValue(req.Value)
	if err != nil {
		return err
	}
	if req.Many {
		docs, ok := doc.(bson.A)
		if !ok {
			return fmt.Errorf("insert many needs an array of documents in the value box")
		}
		inserted, err := collection.InsertMany(ctx, docs)
		if err != nil {
			return fmt.Errorf("insertMany: %w", err)
		}
		res.Affected = int64(len(inserted.InsertedIDs))
		res.Message = fmt.Sprintf("inserted %d document(s)", len(inserted.InsertedIDs))
		return nil
	}
	one, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("insertOne: %w", err)
	}
	res.Affected = 1
	res.Columns = []string{"insertedId"}
	res.Rows = [][]string{{jsonValue(one.InsertedID)}}
	res.RowsReturned = 1
	res.Message = "inserted 1 document"
	return nil
}

func update(ctx context.Context, collection *mongo.Collection, req entity.MongoRequest, res *entity.Result) error {
	filter, err := parseFilter(req.Filter)
	if err != nil {
		return err
	}
	value, err := parseValue(req.Value)
	if err != nil {
		return err
	}
	if req.Many {
		result, err := collection.UpdateMany(ctx, filter, value)
		if err != nil {
			return fmt.Errorf("updateMany: %w", err)
		}
		res.Affected = result.ModifiedCount
		res.Message = fmt.Sprintf("matched %d, modified %d", result.MatchedCount, result.ModifiedCount)
		return nil
	}
	result, err := collection.UpdateOne(ctx, filter, value)
	if err != nil {
		return fmt.Errorf("updateOne: %w", err)
	}
	res.Affected = result.ModifiedCount
	res.Message = fmt.Sprintf("matched %d, modified %d", result.MatchedCount, result.ModifiedCount)
	return nil
}

func remove(ctx context.Context, collection *mongo.Collection, req entity.MongoRequest, res *entity.Result) error {
	filter, err := parseFilter(req.Filter)
	if err != nil {
		return err
	}
	if req.Many {
		result, err := collection.DeleteMany(ctx, filter)
		if err != nil {
			return fmt.Errorf("deleteMany: %w", err)
		}
		res.Affected = result.DeletedCount
		res.Message = fmt.Sprintf("deleted %d document(s)", result.DeletedCount)
		return nil
	}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("deleteOne: %w", err)
	}
	res.Affected = result.DeletedCount
	res.Message = fmt.Sprintf("deleted %d document(s)", result.DeletedCount)
	return nil
}

// renderCursor turns documents into the shared grid: the columns are the union of top-level keys, values are Extended JSON.
func renderCursor(ctx context.Context, cursor *mongo.Cursor, res *entity.Result) error {
	defer func() { _ = cursor.Close(ctx) }()
	docs := make([]bson.M, 0, maxGridRows)
	seen := map[string]bool{}
	columns := make([]string, 0, 16)
	for cursor.Next(ctx) && len(docs) < maxGridRows {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode document: %w", err)
		}
		docs = append(docs, doc)
		for key := range doc {
			if !seen[key] && len(columns) < maxGridColumns {
				seen[key] = true
				columns = append(columns, key)
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	sort.Strings(columns)
	res.Columns = columns
	res.Rows = make([][]string, 0, len(docs))
	// the documents ride along as Extended JSON so the console can render them as JSON or as a tree without guessing from the grid
	res.Documents = make([]string, 0, len(docs))
	for _, doc := range docs {
		row := make([]string, len(columns))
		for i, column := range columns {
			row[i] = jsonValue(doc[column])
		}
		res.Rows = append(res.Rows, row)
		if encoded, err := bson.MarshalExtJSON(doc, false, false); err == nil {
			res.Documents = append(res.Documents, string(encoded))
		}
	}
	res.RowsReturned = len(res.Rows)
	res.Message = fmt.Sprintf("%d document(s)", len(res.Rows))
	return nil
}

// describeAll fills in the stats of every collection, with bounded workers.
func describeAll(ctx context.Context, database *mongo.Database, collections []entity.Collection) []entity.Collection {
	out := make([]entity.Collection, len(collections))
	sem := make(chan struct{}, collStatWorkers)
	done := make(chan int, len(collections))
	for i, collection := range collections {
		go func(i int, name string) {
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = describeOne(ctx, database, name)
			done <- i
		}(i, collection.Name)
	}
	for range collections {
		<-done
	}
	return out
}

// describeOne asks for one collection's stats; a refused collection keeps its name so the list is still usable.
func describeOne(ctx context.Context, database *mongo.Database, name string) entity.Collection {
	out := entity.Collection{Name: name, Type: "collection"}
	var stats bson.M
	if err := database.RunCommand(ctx, bson.D{{Key: "collStats", Value: name}}).Decode(&stats); err != nil {
		return out
	}
	out.Count = number(stats["count"])
	out.SizeBytes = number(stats["size"])
	out.StorageBytes = number(stats["storageSize"])
	out.AvgObjectSize = number(stats["avgObjSize"])
	out.Indexes = number(stats["nindexes"])
	out.IndexSizeBytes = number(stats["totalIndexSize"])
	if capped, ok := stats["capped"].(bool); ok {
		out.Capped = capped
	}
	return out
}

func parseFilter(text string) (bson.M, error) {
	if strings.TrimSpace(text) == "" {
		return bson.M{}, nil
	}
	var doc bson.M
	if err := bson.UnmarshalExtJSON([]byte(text), true, &doc); err != nil {
		return nil, fmt.Errorf("filter is not valid JSON: %w", err)
	}
	return doc, nil
}

// parseValue keeps the update operator document intact ({"$set": {...}}), and accepts a plain document for inserts.
func parseValue(text string) (any, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("value is required")
	}
	if strings.HasPrefix(trimmed, "[") {
		var array bson.A
		if err := bson.UnmarshalExtJSON([]byte(trimmed), true, &array); err != nil {
			return nil, fmt.Errorf("value is not valid JSON: %w", err)
		}
		return array, nil
	}
	var doc bson.M
	if err := bson.UnmarshalExtJSON([]byte(trimmed), true, &doc); err != nil {
		return nil, fmt.Errorf("value is not valid JSON: %w", err)
	}
	return doc, nil
}

// withClient opens a client for one operation and closes it; the caller's context rides along so spans and cancellation keep working.
func withClient(ctx context.Context, c entity.Connection, fn func(ctx context.Context, cl *mongo.Client) error) error {
	client, err := connect(ctx, c)
	if err != nil {
		return err
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	return fn(ctx, client)
}

func jsonValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case bson.ObjectID:
		return typed.Hex()
	case []byte:
		return string(typed)
	}
	encoded, err := bson.MarshalExtJSON(value, false, false)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func stringOf(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

// number reads the numeric fields bson decodes into (int32/int64/float64).
func number(value any) int64 {
	switch typed := value.(type) {
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	}
	return 0
}
