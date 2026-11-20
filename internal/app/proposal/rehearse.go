package proposal

import (
	"context"
	"fmt"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
)

// Rehearse re-runs the proof for a proposal that already exists — because the schema moved since,
// or because the first attempt failed for a reason that has been fixed.
type Rehearse struct{ Deps }

// NewRehearse builds the use case.
func NewRehearse(d Deps) *Rehearse { return &Rehearse{Deps: d} }

// RehearseInput names the proposal and, optionally, a new tier.
type RehearseInput struct {
	Ref  string
	Tier string
}

// Run refuses to re-rehearse something a human already decided: evidence behind an approval that
// has been given is not the same evidence the reviewer saw.
func (r *Rehearse) Run(ctx context.Context, in RehearseInput) (out0 domain.Proposal, out1 domain.Rehearsal, err error) {
	ctx, end := telemetry.Span(ctx, "proposal.Rehearse.Run")
	defer func() { end(err) }()

	value, err := requireRef(in.Ref)
	if err != nil {
		return domain.Proposal{}, domain.Rehearsal{}, err
	}
	proposal, err := r.Proposals.ProposalByRef(ctx, value)
	if err != nil {
		return domain.Proposal{}, domain.Rehearsal{}, err
	}
	switch proposal.Status {
	case domain.StatusDraft, domain.StatusFailed, domain.StatusVerified:
	default:
		return proposal, domain.Rehearsal{}, fmt.Errorf("%w: proposal is %s, and its rehearsal is no longer the thing being judged",
			domain.ErrConflict, proposal.Status)
	}
	if in.Tier != "" {
		if !domain.IsTier(in.Tier) {
			return proposal, domain.Rehearsal{}, fmt.Errorf("%w: tier %q is not one of static, shadow, rehearsal", domain.ErrInvalid, in.Tier)
		}
		proposal.Tier = domain.Tier(in.Tier)
	}
	return r.rehearse(ctx, proposal)
}
