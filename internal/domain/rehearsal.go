package domain

import "time"

// RehearsalStatus is how convincing the rehearsal was. `partial` and `unsupported` exist so a
// route never has to round a weak result up to `verified`.
type RehearsalStatus string

// Rehearsal statuses.
const (
	RehearsalVerified    RehearsalStatus = "verified"
	RehearsalPartial     RehearsalStatus = "partial"
	RehearsalFailed      RehearsalStatus = "failed"
	RehearsalUnsupported RehearsalStatus = "unsupported"
)

// Rehearsal is the evidence behind a proposal: what changed, whether the rollback brought it back,
// and — in the note — exactly how far the claim goes.
type Rehearsal struct {
	Ref          string
	ProposalRef  string
	Tier         Tier
	Status       RehearsalStatus
	SchemaDiff   string
	RowDiff      string
	Plan         string
	Irreversible bool
	Note         string
	DurationMS   int64
	Error        string
	CreatedAt    time.Time
}

// Conclusive reports whether the rehearsal proves the change and its reversal.
func (r Rehearsal) Conclusive() bool { return r.Status == RehearsalVerified }
