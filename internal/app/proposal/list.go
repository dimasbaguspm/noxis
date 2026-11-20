package proposal

import (
	"context"
	"fmt"
	"noxis/internal/common/cursor"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

// List answers one page of the queue, newest first.
type List struct{ Deps }

// NewList builds the use case.
func NewList(d Deps) *List { return &List{Deps: d} }

// ListInput is one page request. Mine is the caller, never a query parameter: a token cannot ask
// for somebody else's proposals by typing a name.
type ListInput struct {
	Cursor string
	Limit  int
	Target string
	Status string
	Kind   string
	Mine   string
}

// ListOutput is one page and the cursor that follows it.
type ListOutput struct {
	Proposals  []domain.Proposal
	NextCursor string
}

// Run clamps the page and validates the filters that name an enum.
func (l *List) Run(ctx context.Context, in ListInput) (ListOutput, error) {
	after, err := cursor.Ref(in.Cursor)
	if err != nil {
		return ListOutput{}, fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	if in.Status != "" && !knownStatus(in.Status) {
		return ListOutput{}, fmt.Errorf("%w: status %q is not one this build knows", domain.ErrInvalid, in.Status)
	}
	if in.Kind != "" && in.Kind != string(domain.KindDML) && in.Kind != string(domain.KindDDL) {
		return ListOutput{}, fmt.Errorf("%w: kind %q is neither dml nor ddl", domain.ErrInvalid, in.Kind)
	}
	limit := cursor.Limit(in.Limit, constants.DefaultPageLimit, constants.MaxPageLimit)

	rows, err := l.Proposals.ListProposals(ctx, ports.ProposalFilter{
		Cursor: after, Limit: limit + 1, Target: in.Target, Status: in.Status, Kind: in.Kind, Mine: in.Mine,
	})
	if err != nil {
		return ListOutput{}, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		return ListOutput{Proposals: rows, NextCursor: cursor.Next(rows[len(rows)-1].Ref, true)}, nil
	}
	return ListOutput{Proposals: rows}, nil
}

func knownStatus(status string) bool {
	for _, known := range []domain.Status{
		domain.StatusDraft, domain.StatusVerified, domain.StatusFailed, domain.StatusApproved,
		domain.StatusRejected, domain.StatusApplying, domain.StatusApplied,
	} {
		if string(known) == status {
			return true
		}
	}
	return false
}
