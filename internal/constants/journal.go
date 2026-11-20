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
