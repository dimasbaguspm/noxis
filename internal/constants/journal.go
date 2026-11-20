package constants

// Subjects a journal row can be about.
const (
	SubjectTarget   = "target"
	SubjectProposal = "proposal"
)

// Journal kinds. The list grows as transitions are added; a kind is never reused for a
// second meaning, because the journal is the audit trail.
const (
	JournalCreated  = "created"
	JournalUpdated  = "updated"
	JournalArchived = "archived"
	JournalProbed   = "probed"
)

// Journal kinds for the proposal lifecycle.
const (
	JournalProposed  = "proposed"
	JournalRehearsed = "rehearsed"
)

// Journal kinds for a decision and an application. An override is its own kind so a reviewer can
// see when separation of duties was set aside, and by whom.
const (
	JournalApproved    = "approved"
	JournalRejected    = "rejected"
	JournalApplied     = "applied"
	JournalApplyFailed = "apply_failed"
	JournalOverride    = "override"
)
