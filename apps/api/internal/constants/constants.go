// Package constants holds universal defined values shared across packages.
package constants

import "time"

// Version is the build version reported by the CLI and health surfaces.
const Version = "0.1.0"

// APIPrefix is the versioned mount point of the public API.
const APIPrefix = "/external/v1"

// AccessGrantTTL is how long every per-connection grant lasts — read, write and admin alike. The single super admin never expires.
const AccessGrantTTL = 120 * 24 * time.Hour

// Access-event actions: the append-only audit trail.
const (
	AccessGranted   = "granted"
	AccessRevoked   = "revoked"
	AccessRequested = "requested"
	AccessApproved  = "approved"
	AccessRejected  = "rejected"
)

// The wire contract ops (SQL console operations).
const (
	OpQuery    = "query"
	OpAlter    = "alter"
	OpRollback = "rollback"
	OpAnalyze  = "analyze"
)

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
	DriverMongo  = "mongo"
	// DriverPostgres and its siblings are their own engine: the API namespace, the registry and the change queue are keyed on the driver name.
)

// Driver capabilities are what a driver's module can actually do, so the console groups its navigation by capability and a driver it has never seen still appears under the group it serves.
const (
	// CapabilitySQL is querying, reading the schema and proving a SQL change.
	CapabilitySQL = "sql"
	// CapabilityKeys is browsing and removing memory-cache keys.
	CapabilityKeys = "keys"
	// CapabilityDocuments is browsing documents and submitting a document request.
	CapabilityDocuments = "documents"
)

// SchedulerActor attributes changes applied by the authenticated scheduler callback.
const SchedulerActor = "system:scheduler"

// MongoDB request ops.
const (
	MongoFind      = "find"
	MongoAggregate = "aggregate"
	MongoCount     = "count"
	MongoInsert    = "insert"
	MongoUpdate    = "update"
	MongoDelete    = "delete"
)

// Change kinds (which statement family a change carries).
const (
	KindDML   = "dml"
	KindDDL   = "ddl"
	KindMongo = "mongo" // a MongoDB request, reviewed like a DML change
	// KindDocument is a document write on a store that is not MongoDB, reviewed the same way.
	KindDocument = "document"
	// KindSearch is a change on a search platform: an index, a document, a saved search.
	KindSearch = "search"
)

// Change workflow statuses.
const (
	ChangeValidated  = "validated"  // query and rollback dry-ran clean, together
	ChangeUnverified = "unverified" // driver cannot dry-run it (mysql DDL) — approve with eyes open
	ChangeFailed     = "failed"     // validation found a problem
	ChangeApproved   = "approved"   // a human signed off (self-approval allowed)
	ChangeApplying   = "applying"   // one caller claimed execution; outcome may be unknown after a crash
	ChangeApplied    = "applied"    // executed and committed
	ChangeRejected   = "rejected"   // reviewed and declined (kept as an artifact)
)

// Unverified reasons: a closed set, so a reviewer reads one line and the queue can group what no driver could prove.
const (
	// UnverifiedServerDDL is a statement that changes the server rather than a schema.
	UnverifiedServerDDL = "server-ddl"
	// UnverifiedForeignSchema is a name in a database the scratch one cannot stand in for.
	UnverifiedForeignSchema = "foreign-schema"
	// UnverifiedQualifiedDDL is a qualified name a scratch schema would not contain.
	UnverifiedQualifiedDDL = "qualified-ddl"
	// UnverifiedNoRows is a DML change whose tables held no sampled rows, so a rollback proved nothing.
	UnverifiedNoRows = "no-rows"
)
