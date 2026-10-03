// Package constants holds universal defined values shared across modules.
// Domain strings stay with their module (internal/sql/constants.go);
// user-facing error messages live next to the logic that raises them.
package constants

// Version is the build version reported by the CLI and health surfaces.
const Version = "0.1.0"

// Change workflow statuses.
const (
	ChangeValidated  = "validated"  // query and rollback dry-ran clean, together
	ChangeUnverified = "unverified" // driver cannot dry-run it (mysql DDL) — approve with eyes open
	ChangeFailed     = "failed"     // validation found a problem
	ChangeApproved   = "approved"   // a human signed off (self-approval allowed)
	ChangeApplied    = "applied"    // executed and committed
)
