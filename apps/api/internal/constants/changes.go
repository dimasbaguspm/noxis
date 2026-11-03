package constants

// Change kinds (which statement family a change carries).
const (
	KindDML = "dml"
	KindDDL = "ddl"
	// KindMongo is a MongoDB request, reviewed like a DML change.
	KindMongo = "mongo"
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

// SchedulerActor attributes changes applied by the authenticated scheduler callback.
const SchedulerActor = "system:scheduler"
