package constants

// The wire contract ops (SQL console operations).
const (
	OpQuery    = "query"
	OpAlter    = "alter"
	OpRollback = "rollback"
	OpAnalyze  = "analyze"
)

// Search-platform ops: a read, and the one write a search platform can review.
const (
	SearchOpSearch = "search"
	SearchOpDelete = "delete"
)

// MongoDB request ops.
const (
	MongoFind      = "find"
	MongoAggregate = "aggregate"
	MongoCount     = "count"
	MongoInsert    = "insert"
	MongoUpdate    = "update"
	MongoDelete    = "delete"
)
