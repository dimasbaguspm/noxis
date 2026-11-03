package constants

// Supported database drivers (mariadb is served by the mysql driver).
const (
	DriverPostgres = "postgres"
	DriverMySQL    = "mysql"
	// DriverMariaDB is MariaDB: the same wire protocol as mysql, a different server, so it is its own driver name.
	DriverMariaDB = "mariadb"
	// DriverSQLite is a file-backed SQL database: the connection's Database is the file path, with no host, port or user.
	DriverSQLite = "sqlite"
	// DriverCockroach is a postgres-wire SQL database with transactional DDL: a change is proved in the live transaction, never in a throwaway schema.
	DriverCockroach = "cockroach"
	DriverRedis     = "redis"
	// DriverValkey is the Linux Foundation fork of Redis: same protocol, same commands. It is a separate driver name so the console can show what it is actually talking to.
	DriverValkey = "valkey"
	// DriverDragonfly is a redis-protocol cache server: the same commands, its own driver name so the console shows what
	// it is actually talking to.
	DriverDragonfly = "dragonfly"
	DriverMongo     = "mongo"
	// DriverCouchDB is a document store with a JSON API: the console's document seam fits it, and a write is proved in a
	// throwaway database rather than a scratch collection.
	DriverCouchDB = "couchdb"
	// DriverElasticsearch is a search platform: it indexes documents and answers its own query language, so it is
	// neither a SQL engine nor a document store.
	DriverElasticsearch = "elasticsearch"
	// DriverMeilisearch is a search platform with its own query parameters and an asynchronous write API.
	DriverMeilisearch = "meilisearch"
	// DriverSplunk is a search and observability deployment: its query language is SPL and its results are events.
	DriverSplunk = "splunk"
)

// Driver capabilities are what a driver's module can actually do, so the console groups its navigation by capability and a driver it has never seen still appears under the group it serves.
const (
	// CapabilitySQL is querying, reading the schema and proving a SQL change.
	CapabilitySQL = "sql"
	// CapabilityKeys is browsing and removing memory-cache keys.
	CapabilityKeys = "keys"
	// CapabilityDocuments is browsing documents and submitting a document request.
	CapabilityDocuments = "documents"
	// CapabilitySearch is browsing what a search platform indexes, querying it, and reviewing a change.
	CapabilitySearch = "search"
)
