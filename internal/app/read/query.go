package read

import (
	"context"
	"fmt"
	"noxis/internal/common/cursor"
	"noxis/internal/common/sqltext"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

// Query runs one statement after the lexer says it is a read. It is the only route that takes SQL
// from a caller, and the only one that must refuse rather than pass a mistake through.
type Query struct{ Deps }

// NewQuery builds the use case.
func NewQuery(d Deps) *Query { return &Query{Deps: d} }

// QueryInput is one statement and its ceiling.
type QueryInput struct {
	Ref   string
	SQL   string
	Limit int
}

// Run classifies, refuses, connects and reads. The refusal happens before a connection is opened,
// so a write never reaches an engine even when the target is reachable.
func (q *Query) Run(ctx context.Context, in QueryInput) (out0 domain.Rows, err error) {
	ctx, end := telemetry.Span(ctx, "read.Query.Run")
	defer func() { end(err) }()

	statement, err := sqltext.Classify(in.SQL)
	if err != nil {
		return domain.Rows{}, fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	if !statement.Read() {
		return domain.Rows{}, fmt.Errorf("%w: a read runs SELECT, EXPLAIN or SHOW; %q is %s",
			domain.ErrInvalid, statement.Kind, statement.Kind)
	}
	if statement.Locking {
		return domain.Rows{}, fmt.Errorf("%w: %s", domain.ErrInvalid, statement.Reason)
	}

	target, password, err := q.connect(ctx, in.Ref)
	if err != nil {
		return domain.Rows{}, err
	}
	limit := cursor.Limit(in.Limit, constants.DefaultRowsLimit, constants.MaxRowsLimit)

	ctx, cancel := context.WithTimeout(ctx, q.Timeout)
	defer cancel()
	return q.Driver.Query(ctx, target, password, statement.Raw, limit)
}
