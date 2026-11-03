package constants

import "time"

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
