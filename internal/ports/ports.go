// Package ports declares what the use cases need, in the use cases' own vocabulary.
// The interfaces are consumer-owned: an adapter implements them, it does not define them,
// and nothing here mentions SQL, HTTP or a driver.
package ports

import (
	"context"

	"noxis/internal/domain"
)

// Targets is the store of governed servers.
type Targets interface {
	Create(ctx context.Context, target domain.Target) (domain.Target, error)
	ByRef(ctx context.Context, ref string) (domain.Target, error)
	List(ctx context.Context, filter TargetFilter) ([]domain.Target, error)
	Update(ctx context.Context, target domain.Target) (domain.Target, error)
}

// TargetFilter is one page request: a keyset cursor, a ceiling, and the filters a human
// actually types. It never carries an offset.
type TargetFilter struct {
	Cursor          string
	Limit           int
	Driver          string
	Query           string
	IncludeArchived bool
}

// Journal is the append-only audit trail.
type Journal interface {
	Append(ctx context.Context, entries ...domain.JournalEntry) error
	Entries(ctx context.Context, subjectRef string, limit int) ([]domain.JournalEntry, error)
}

// Sealer seals a target password at rest. internal/common/secret implements it, with the one key
// the deployment was given.
type Sealer interface {
	Enabled() bool
	Seal(aad string, plain []byte) (string, error)
	Open(aad, sealed string) ([]byte, error)
}

// Driver is one engine this build speaks. It answers only what an engine can answer: it
// never stores, never decides policy, and never sees a sealed password.
type Driver interface {
	Probe(ctx context.Context, target domain.Target, password string) (domain.Capabilities, error)
	// Query runs one statement a read route has already classified, and returns at most limit
	// rows plus whether it stopped early.
	Query(ctx context.Context, target domain.Target, password, statement string, limit int) (domain.Rows, error)
	// Databases lists the schemas the connection can see.
	Databases(ctx context.Context, target domain.Target, password string) ([]string, error)
	// Tables lists the tables in one schema.
	Tables(ctx context.Context, target domain.Target, password, database string) ([]domain.Table, error)
	// Table describes one table: its columns, its indexes and the DDL the engine would use to
	// create it, which is the same statement a rehearsal will rebuild from.
	Table(ctx context.Context, target domain.Target, password, database, table string) (domain.TableDetail, error)
	// Rehearse proves a change in a throwaway database rebuilt from the engine's own DDL. It
	// returns a verdict rather than an error for a change that does not survive: the caller stores
	// the evidence either way.
	Rehearse(ctx context.Context, target domain.Target, password string, change domain.Change) (domain.Rehearsal, error)
	// Plan says how big the change is before it runs, so the caller can be told to poll instead of
	// holding a request open for a table copy.
	Plan(ctx context.Context, target domain.Target, password string, change domain.Change) (domain.Plan, error)
	// Apply runs the approved statement for real, in one transaction where the engine has one.
	// report receives what the engine says about the statement while it runs; it may be nil.
	Apply(ctx context.Context, target domain.Target, password string, change domain.Change,
		report func(domain.Progress)) (domain.Receipt, error)
}

// Proposals is the store of proposed changes. A proposal is never deleted: a rejected one is a
// decision someone may need to read again.
type Proposals interface {
	CreateProposal(ctx context.Context, proposal domain.Proposal) (domain.Proposal, error)
	ProposalByRef(ctx context.Context, ref string) (domain.Proposal, error)
	ListProposals(ctx context.Context, filter ProposalFilter) ([]domain.Proposal, error)
	UpdateProposal(ctx context.Context, proposal domain.Proposal) (domain.Proposal, error)
	// ClaimProposal moves a proposal from one status to another only while it is still in `from`.
	// Apply uses it before touching a database, so two operators cannot both believe they won.
	ClaimProposal(ctx context.Context, ref string, from, to domain.Status) (domain.Proposal, error)
	// SetProgress writes only what the engine said about the running statement, so the watcher never
	// races the receipt that ends the apply.
	SetProgress(ctx context.Context, ref string, progress domain.Progress) error
}

// ProposalFilter is one page of the queue: a keyset cursor, a ceiling, and the filters a reviewer
// types. It never carries an offset.
type ProposalFilter struct {
	Cursor string
	Limit  int
	Target string
	Status string
	Kind   string
	Mine   string
}

// Rehearsals is the history of attempts to prove one proposal. Every run is stored, including the
// ones that failed: "we tried and it did not prove" is a fact the reviewer needs.
type Rehearsals interface {
	CreateRehearsal(ctx context.Context, rehearsal domain.Rehearsal) (domain.Rehearsal, error)
	RehearsalByRef(ctx context.Context, ref string) (domain.Rehearsal, error)
	LatestRehearsal(ctx context.Context, proposalRef string) (domain.Rehearsal, error)
}
