// Package constants holds universal defined values shared across packages.
// Domain behaviour (statement kinds, preview bounds) stays with its service;
// user-facing error messages live next to the logic that raises them.
package constants

import "strings"

// Version is the build version reported by the CLI and health surfaces.
const Version = "0.1.0"

// The wire contract ops (SQL console operations).
const (
	OpQuery     = "query"
	OpAlter     = "alter"
	OpRollback  = "rollback"
	OpAnalytics = "analytics"
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
)

// Engine families: the API namespace and console section that serve a
// connection's driver. Each family owns its own connection list, even though
// both write to the same store.
const (
	FamilySQL   = "sql"
	FamilyRedis = "redis"
)

// FamilyOf maps a driver to the family that serves it.
func FamilyOf(driver string) string {
	if strings.EqualFold(driver, DriverRedis) || strings.EqualFold(driver, DriverValkey) {
		return FamilyRedis
	}
	return FamilySQL
}

// Change kinds (which statement family a change carries).
const (
	KindDML = "dml"
	KindDDL = "ddl"
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
