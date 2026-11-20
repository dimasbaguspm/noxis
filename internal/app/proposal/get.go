package proposal

import (
	"context"
	"noxis/internal/domain"
)

// Get reads one proposal and the rehearsal that last judged it.
type Get struct{ Deps }

// NewGet builds the use case.
func NewGet(d Deps) *Get { return &Get{Deps: d} }

// GetOutput carries the proposal with its evidence, so a reviewer never has to make two calls to
// see one decision.
type GetOutput struct {
	Proposal     domain.Proposal
	Rehearsal    domain.Rehearsal
	HasRehearsal bool
}

// Run reads the proposal, then its latest rehearsal when there is one.
func (g *Get) Run(ctx context.Context, ref string) (GetOutput, error) {
	value, err := requireRef(ref)
	if err != nil {
		return GetOutput{}, err
	}
	proposal, err := g.Proposals.ProposalByRef(ctx, value)
	if err != nil {
		return GetOutput{}, err
	}
	rehearsal, err := g.Rehearsals.LatestRehearsal(ctx, proposal.Ref)
	if err != nil {
		// A proposal with no rehearsal is a proposal that could not be attempted: not an error.
		return GetOutput{Proposal: proposal}, nil
	}
	return GetOutput{Proposal: proposal, Rehearsal: rehearsal, HasRehearsal: true}, nil
}
