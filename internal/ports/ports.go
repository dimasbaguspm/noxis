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
}
