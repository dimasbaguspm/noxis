package proposal

import (
	"context"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
)

// Events reads the history of one proposal: proposed, rehearsed, decided, applied. The journal is
// append-only, so this list only ever grows.
type Events struct{ Deps }

// NewEvents builds the use case.
func NewEvents(d Deps) *Events { return &Events{Deps: d} }

// Run answers the journal lines for one proposal, newest first.
func (e *Events) Run(ctx context.Context, ref string) (out0 []domain.JournalEntry, err error) {
	ctx, end := telemetry.Span(ctx, "proposal.Events.Run")
	defer func() { end(err) }()

	value, err := requireRef(ref)
	if err != nil {
		return nil, err
	}
	if _, err := e.Proposals.ProposalByRef(ctx, value); err != nil {
		return nil, err
	}
	return e.Journal.Entries(ctx, value, 200)
}
