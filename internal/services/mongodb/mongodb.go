// Package mongodb explores and queries a MongoDB server: databases and
// collections with their sizes, CRUD requests with separate filter and value
// documents, and the metrics worth watching (server status, db and collection
// stats).
//
// ponytail: requests run directly, with no validated rollback like the SQL
// change queue has. A document change cannot be dry-run generically, so the
// guards here are the honest ones: an op allowlist, and no update or delete
// without a filter.
package mongodb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const (
	pingTimeout = 10 * time.Second
	opTimeout   = 30 * time.Second

	// maxGridColumns bounds a rendered document grid: the union of keys across
	// the returned documents is capped so one wide collection cannot produce a
	// thousand-column table.
	maxGridColumns = 40

	// maxGridRows bounds how many documents a read request renders.
	maxGridRows = 200

	// collStatWorkers bounds how many collections are described at once.
	collStatWorkers = 4
)

// DB talks to MongoDB servers.
type DB struct{}

func New() *DB { return &DB{} }

// Result is the shared operation outcome.
type Result = entity.Result

// Database is one database with its size, as `dbStats` reports it.
type Database struct {
	Name           string `json:"name"`
	SizeBytes      int64  `json:"sizeBytes"`      // uncompressed data
	StorageBytes   int64  `json:"storageBytes"`   // on-disk
	IndexSizeBytes int64  `json:"indexSizeBytes"` //
	Collections    int64  `json:"collections"`
	Objects        int64  `json:"objects"`
	AvgObjectSize  int64  `json:"avgObjectSizeBytes"`
}

// Collection is one collection with its size, as `collStats` reports it.
type Collection struct {
	Name           string `json:"name"`
	Type           string `json:"type"` // collection | view | timeseries
	Count          int64  `json:"count"`
	SizeBytes      int64  `json:"sizeBytes"`
	StorageBytes   int64  `json:"storageBytes"`
	AvgObjectSize  int64  `json:"avgObjectSizeBytes"`
	Indexes        int64  `json:"indexes"`
	IndexSizeBytes int64  `json:"indexSizeBytes"`
	Capped         bool   `json:"capped"`
}

// Request is one CRUD request: the filter and the value are separate
// documents, both written as MongoDB Extended JSON.
type Request struct {
	Op         string `json:"op"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter,omitempty"`
	Value      string `json:"value,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Limit      int64  `json:"limit,omitempty"`
	Many       bool   `json:"many,omitempty"`
}

// Stats is one analytics snapshot, focused on a database and collection.
type Stats struct {
	Database   string `json:"database"`
	Collection string `json:"collection"`

	// server
	Version         string `json:"version"`
	UptimeSeconds   int64  `json:"uptimeSeconds"`
	Connections     int64  `json:"connections"`
	CurrentOps      int64  `json:"currentOps"`
	Inserted        int64  `json:"inserted"`
	Queried         int64  `json:"queried"`
	Updated         int64  `json:"updated"`
	Deleted         int64  `json:"deleted"`
	ResidentMB      int64  `json:"residentMemoryMb"`
	VirtualMB       int64  `json:"virtualMemoryMb"`
	NetworkInBytes  int64  `json:"networkInBytes"`
	NetworkOutBytes int64  `json:"networkOutBytes"`

	// database
	Collections    int64 `json:"dbCollections"`
	Objects        int64 `json:"dbObjects"`
	DataSizeBytes  int64 `json:"dbDataSizeBytes"`
	StorageBytes   int64 `json:"dbStorageBytes"`
	Indexes        int64 `json:"dbIndexes"`
	IndexSizeBytes int64 `json:"dbIndexSizeBytes"`

	// collection (zero when none was asked for)
	Count            int64 `json:"collectionCount"`
	CollSizeBytes    int64 `json:"collectionSizeBytes"`
	CollStorageBytes int64 `json:"collectionStorageBytes"`
	CollAvgObjSize   int64 `json:"collectionAvgObjectSizeBytes"`
	CollIndexes      int64 `json:"collectionIndexes"`
	CollIndexBytes   int64 `json:"collectionIndexSizeBytes"`
	Capped           bool  `json:"collectionCapped"`
}

// Family is the namespace this module serves.
func (m *DB) Family() string { return constants.FamilyMongo }

// Supports reports whether this module serves the driver.
func (m *DB) Supports(driver string) bool {
	return strings.EqualFold(driver, constants.DriverMongo) || strings.EqualFold(driver, "mongodb")
}

// Ping verifies the server answers and the credentials work.
func (m *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
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
func (m *DB) Databases(ctx context.Context, c entity.Connection) (out []Database, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		names, err := cl.ListDatabaseNames(ctx, bson.D{})
		if err != nil {
			return fmt.Errorf("list databases on %s: %w", c.Name, err)
		}
		out = make([]Database, 0, len(names))
		for _, name := range names {
			db := Database{Name: name}
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
func (m *DB) Collections(ctx context.Context, c entity.Connection, database string) (out []Collection, err error) {
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

		var described []Collection
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return fmt.Errorf("decode collection: %w", err)
			}
			collection := Collection{Name: stringOf(doc["name"]), Type: stringOf(doc["type"])}
			if collection.Type == "" {
				collection.Type = "collection"
			}
			described = append(described, collection)
		}
		if err := cursor.Err(); err != nil {
			return fmt.Errorf("iterate collections: %w", err)
		}

		// collStats is one round trip per collection; bounded workers keep a
		// wide database from serialising.
		out = describeAll(ctx, cl.Database(database), described)
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return nil
	})
	return out, err
}

// Run executes one CRUD request.
func (m *DB) Run(ctx context.Context, c entity.Connection, req Request) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	started := time.Now()
	defer func() { res.DurationMs = time.Since(started).Milliseconds() }()

	if err := req.validate(); err != nil {
		return Result{}, err
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

// Stats reads server status, database stats and (when asked) collection stats.
func (m *DB) Stats(ctx context.Context, c entity.Connection, database, collection string) (stats Stats, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	err = withClient(ctx, c, func(ctx context.Context, cl *mongo.Client) error {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		stats.Database, stats.Collection = database, collection

		var server bson.M
		if err := cl.Database("admin").RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&server); err != nil {
			return fmt.Errorf("serverStatus: %w", err)
		}
		stats.Version = stringOf(server["version"])
		stats.UptimeSeconds = number(server["uptime"])
		// serverStatus nests documents, and the driver hands those back as
		// bson.D rather than bson.M, so each lookup goes through docOf.
		if connections := docOf(server["connections"]); connections != nil {
			stats.Connections = number(connections["current"])
		}
		if ops := docOf(server["opcounters"]); ops != nil {
			stats.Inserted = number(ops["insert"])
			stats.Queried = number(ops["query"])
			stats.Updated = number(ops["update"])
			stats.Deleted = number(ops["delete"])
		}
		if metrics := docOf(server["metrics"]); metrics != nil {
			if operations := docOf(metrics["operation"]); operations != nil {
				stats.CurrentOps = number(operations["current"])
			}
		}
		if memory := docOf(server["mem"]); memory != nil {
			stats.ResidentMB = number(memory["resident"])
			stats.VirtualMB = number(memory["virtual"])
		}
		if network := docOf(server["network"]); network != nil {
			stats.NetworkInBytes = number(network["bytesIn"])
			stats.NetworkOutBytes = number(network["bytesOut"])
		}

		if database == "" {
			return nil
		}
		var dbStats bson.M
		if err := cl.Database(database).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStats); err != nil {
			return fmt.Errorf("dbStats %s: %w", database, err)
		}
		stats.Collections = number(dbStats["collections"])
		stats.Objects = number(dbStats["objects"])
		stats.DataSizeBytes = number(dbStats["dataSize"])
		stats.StorageBytes = number(dbStats["storageSize"])
		stats.Indexes = number(dbStats["indexes"])
		stats.IndexSizeBytes = number(dbStats["indexSize"])

		if collection == "" {
			return nil
		}
		described := describeAll(ctx, cl.Database(database), []Collection{{Name: collection}})
		if len(described) == 0 {
			return fmt.Errorf("collection %s.%s has no stats", database, collection)
		}
		one := described[0]
		stats.Count = one.Count
		stats.CollSizeBytes = one.SizeBytes
		stats.CollStorageBytes = one.StorageBytes
		stats.CollAvgObjSize = one.AvgObjectSize
		stats.CollIndexes = one.Indexes
		stats.CollIndexBytes = one.IndexSizeBytes
		stats.Capped = one.Capped
		return nil
	})
	return stats, err
}

// validate keeps the request honest: a known op, a target, and no update or
// delete without a filter (an empty filter would rewrite or empty the whole
// collection).
func (r Request) validate() error {
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

func find(ctx context.Context, collection *mongo.Collection, req Request, res *Result) error {
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

func aggregate(ctx context.Context, collection *mongo.Collection, req Request, res *Result) error {
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

func insert(ctx context.Context, collection *mongo.Collection, req Request, res *Result) error {
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

func update(ctx context.Context, collection *mongo.Collection, req Request, res *Result) error {
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

func remove(ctx context.Context, collection *mongo.Collection, req Request, res *Result) error {
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

// renderCursor turns documents into the shared grid: the columns are the union
// of top-level keys, values are Extended JSON.
func renderCursor(ctx context.Context, cursor *mongo.Cursor, res *Result) error {
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
	for _, doc := range docs {
		row := make([]string, len(columns))
		for i, column := range columns {
			row[i] = jsonValue(doc[column])
		}
		res.Rows = append(res.Rows, row)
	}
	res.RowsReturned = len(res.Rows)
	res.Message = fmt.Sprintf("%d document(s)", len(res.Rows))
	return nil
}

// describeAll fills in the stats of every collection, with bounded workers.
func describeAll(ctx context.Context, database *mongo.Database, collections []Collection) []Collection {
	out := make([]Collection, len(collections))
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

// describeOne asks for one collection's stats; a refused collection keeps its
// name so the list is still usable.
func describeOne(ctx context.Context, database *mongo.Database, name string) Collection {
	out := Collection{Name: name, Type: "collection"}
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

// parseValue keeps the update operator document intact ({"$set": {...}}), and
// accepts a plain document for inserts.
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

// withClient opens a client for one operation and closes it; the caller's
// context rides along so spans and cancellation keep working.
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

// docOf normalizes a nested document, whichever representation the driver
// chose for it.
func docOf(value any) bson.M {
	switch typed := value.(type) {
	case bson.M:
		return typed
	case bson.D:
		out := make(bson.M, len(typed))
		for _, element := range typed {
			out[element.Key] = element.Value
		}
		return out
	case map[string]any:
		return typed
	}
	return nil
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
