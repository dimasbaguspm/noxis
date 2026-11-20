package target

import (
	"context"
	"fmt"
	"noxis/internal/common/cursor"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

// List answers one page of targets, newest first, keyed by the ref of the last row.
type List struct{ Deps }

// NewList builds the use case.
func NewList(d Deps) *List { return &List{Deps: d} }

// ListInput is one page request.
type ListInput struct {
	Cursor          string
	Limit           int
	Driver          string
	Query           string
	IncludeArchived bool
}

// ListOutput is one page and the cursor that follows it. NextCursor is empty on the last
// page: a full page is the only evidence that more rows exist.
type ListOutput struct {
	Targets    []domain.Target
	NextCursor string
}

// Run clamps the page, validates the filters and asks the store for one row more than the
// page size, which is how "is there a next page" is answered without counting.
func (l *List) Run(ctx context.Context, in ListInput) (ListOutput, error) {
	after, err := cursor.Ref(in.Cursor)
	if err != nil {
		return ListOutput{}, fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	if in.Driver != "" && !constants.IsDriver(in.Driver) {
		return ListOutput{}, fmt.Errorf("%w: driver %q is not one of %v", domain.ErrInvalid, in.Driver, constants.Drivers)
	}
	limit := cursor.Limit(in.Limit, constants.DefaultPageLimit, constants.MaxPageLimit)

	rows, err := l.Targets.List(ctx, ports.TargetFilter{
		Cursor:          after,
		Limit:           limit + 1,
		Driver:          in.Driver,
		Query:           in.Query,
		IncludeArchived: in.IncludeArchived,
	})
	if err != nil {
		return ListOutput{}, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		return ListOutput{Targets: rows, NextCursor: cursor.Next(rows[len(rows)-1].Ref, true)}, nil
	}
	return ListOutput{Targets: rows}, nil
}
