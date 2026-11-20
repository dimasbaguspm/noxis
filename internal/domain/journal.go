package domain

import "time"

// JournalEntry is one append-only line: who did what, to which subject, and when.
// Nothing in this build updates or deletes one.
type JournalEntry struct {
	Ref         string
	SubjectRef  string
	SubjectKind string
	Kind        string
	Actor       string
	Detail      map[string]string
	CreatedAt   time.Time
}
