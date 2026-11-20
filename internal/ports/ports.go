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

// Sealer seals a target password at rest. internal/common/secret implements it; the second
// return of Open reports a value that only a previous key could open.
type Sealer interface {
	Enabled() bool
	Seal(aad string, plain []byte) (string, error)
	Open(aad, sealed string) ([]byte, bool, error)
}
