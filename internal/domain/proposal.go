package domain

import (
	"fmt"
	"strings"
	"time"
)

// Kind is what a proposal changes.
type Kind string

// Proposal kinds: a statement that moves data, or one that moves the schema.
const (
	KindDML Kind = "dml"
	KindDDL Kind = "ddl"
)

// Status is where a proposal stands. The state machine below is the only writer of it.
type Status string

// Proposal statuses.
const (
	StatusDraft    Status = "draft"
	StatusVerified Status = "verified"
	StatusFailed   Status = "failed"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusApplying Status = "applying"
	StatusApplied  Status = "applied"
)

// Tier is how deeply a change was rehearsed: what the reviewer is allowed to assume.
type Tier string

// Tiers, from cheapest to most convincing.
const (
	TierStatic    Tier = "static"
	TierShadow    Tier = "shadow"
	TierRehearsal Tier = "rehearsal"
)

// Tiers is every tier a proposal may ask for.
var Tiers = []Tier{TierStatic, TierShadow, TierRehearsal}

// IsTier reports whether s names a tier this build runs.
func IsTier(s string) bool {
	for _, tier := range Tiers {
		if string(tier) == s {
			return true
		}
	}
	return false
}

// Change is the pair a proposal carries: what to do, and how to undo it. An empty Rollback is
// the honest way to say "there is no way back".
type Change struct {
	Query    string
	Rollback string
	Tier     Tier
}

// Algorithm is how the engine ran a statement over a table, in the engine's own vocabulary. It is
// what the receipt says, because "the change landed" and "the table was rewritten" are different
// facts for whoever reads it next.
type Algorithm string

// The algorithms both engines name.
const (
	AlgorithmInstant Algorithm = "INSTANT"
	AlgorithmInPlace Algorithm = "INPLACE"
	AlgorithmCopy    Algorithm = "COPY"
)

// Locked reports whether the engine had to rebuild the table, which blocks writes while it runs.
func (a Algorithm) Locked() bool { return a == AlgorithmCopy }

// Progress is what the engine says about a statement that is still running. Percent is -1 when the
// engine does not say: MariaDB reports it for a copy, MySQL 8 reports it per stage.
type Progress struct {
	Stage   string
	Percent int
}

// UnknownProgress is progress with no stage and no percentage: the shape a caller gets when the
// statement is either not running or not described.
var UnknownProgress = Progress{Percent: -1}

// Plan is what can be said about a change before it runs: how big the table is, and which algorithm
// the engine is expected to choose.
type Plan struct {
	RowsEstimated int64
	Algorithm     Algorithm
}

// Receipt is what the engine reports after the statement ran.
type Receipt struct {
	Algorithm    Algorithm
	RowsAffected int64
}

// Proposal is one proposed change, rehearsed before a human sees it.
type Proposal struct {
	Ref          string
	TargetRef    string
	Title        string
	Kind         Kind
	Query        string
	Rollback     string
	Note         string
	Status       Status
	Tier         Tier
	Irreversible bool
	ProposedBy   string
	RehearsalRef string
	DecidedBy    string
	DecidedAt    *time.Time
	DecidedNote  string
	AppliedBy    string
	AppliedAt    *time.Time
	Failure      string
	// What the engine said while it ran, and what it did when it finished.
	StartedAt     *time.Time
	Progress      Progress
	Algorithm     Algorithm
	RowsEstimated int64
	RowsAffected  int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	// IdempotencyKey is the caller's name for the request that made this proposal, empty when the
	// caller named none. A change proposed twice under one key is one change.
	IdempotencyKey string
}

// Validate refuses a proposal a reviewer could not act on. The statement itself is judged by the
// lexer, not here: the domain does not read SQL.
func (p Proposal) Validate() error {
	if strings.TrimSpace(p.Title) == "" {
		return fmt.Errorf("%w: a proposal needs a title a reviewer can scan", ErrInvalid)
	}
	if len(p.Title) > 200 {
		return fmt.Errorf("%w: the title is longer than 200 characters", ErrInvalid)
	}
	if strings.TrimSpace(p.Query) == "" {
		return fmt.Errorf("%w: a proposal needs a statement", ErrInvalid)
	}
	if p.Kind != KindDML && p.Kind != KindDDL {
		return fmt.Errorf("%w: kind %q is neither dml nor ddl", ErrInvalid, p.Kind)
	}
	if !IsTier(string(p.Tier)) {
		return fmt.Errorf("%w: tier %q is not one of static, shadow, rehearsal", ErrInvalid, p.Tier)
	}
	if p.ProposedBy == "" {
		return fmt.Errorf("%w: a proposal needs the caller that made it", ErrInvalid)
	}
	if p.TargetRef == "" {
		return fmt.Errorf("%w: a proposal needs a target", ErrInvalid)
	}
	return nil
}

// transitions is the whole state machine. Everything not listed is refused by name.
var transitions = map[Status][]Status{
	StatusDraft:    {StatusVerified, StatusFailed},
	StatusFailed:   {StatusDraft, StatusVerified},
	StatusVerified: {StatusApproved, StatusRejected, StatusFailed},
	StatusApproved: {StatusApplying, StatusRejected},
	StatusApplying: {StatusApplied, StatusFailed},
}

// CanTransition reports whether status may become next.
func (p Proposal) CanTransition(next Status) bool {
	for _, candidate := range transitions[p.Status] {
		if candidate == next {
			return true
		}
	}
	return false
}

// Decidable reports whether a human may still approve or reject it.
func (p Proposal) Decidable() bool {
	return p.Status == StatusVerified || p.Status == StatusApproved
}

// Applyable reports whether the statement may be run for real.
func (p Proposal) Applyable() bool { return p.Status == StatusApproved }
