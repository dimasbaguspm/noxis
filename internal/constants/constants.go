// Package constants holds universal defined values shared across packages.
// Domain behaviour (statement kinds, preview bounds) stays with its service;
// user-facing error messages live next to the logic that raises them.
package constants

// Version is the build version reported by the CLI and health surfaces.
const Version = "0.1.0"

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
	DriverRedis    = "redis"
	// DriverValkey is the Linux Foundation fork of Redis: same protocol, same
	// commands, same family. It is a separate driver name so the console can
	// show what it is actually talking to.
	DriverValkey = "valkey"
	DriverMongo  = "mongo"
)

// Engine families: the API namespace and console section that serve a
// connection's driver. Each family owns its own connection list, even though
// they all write to the same store. A family's module reports its own name
// (interfaces.Module.Family), so no driver-to-family switch lives here.
const (
	FamilySQL      = "sql"
	FamilyMemcache = "memcache"
	FamilyMongo    = "mongodb"
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

// Change kinds (which statement family a change carries).
const (
	KindDML   = "dml"
	KindDDL   = "ddl"
	KindMongo = "mongo" // a MongoDB request, reviewed like a DML change
)

// Change workflow statuses.
const (
	ChangeValidated  = "validated"  // query and rollback dry-ran clean, together
	ChangeUnverified = "unverified" // driver cannot dry-run it (mysql DDL) — approve with eyes open
	ChangeFailed     = "failed"     // validation found a problem
	ChangeApproved   = "approved"   // a human signed off (self-approval allowed)
	ChangeApplied    = "applied"    // executed and committed
	ChangeRejected   = "rejected"   // reviewed and declined (kept as an artifact)
)
