package proposal

import (
	"context"
	"fmt"
	"time"

	"noxis/internal/constants"
	"noxis/internal/domain"
)

// Decide approves or rejects a proposal. It is the one use case where a human's name matters more
// than the statement: the decision, who made it and why all land in the record.
type Decide struct{ Deps }

// NewDecide builds the use case.
func NewDecide(d Deps) *Decide { return &Decide{Deps: d} }

// DecideInput is one decision. ActorIsAdmin is read from the caller's token, never from a body.
type DecideInput struct {
	Ref          string
	Approve      bool
	Note         string
	Actor        string
	ActorIsAdmin bool
}

// Run refuses a decision that separation of duties, or the rehearsal, does not allow.
func (d *Decide) Run(ctx context.Context, in DecideInput) (domain.Proposal, error) {
	ref, err := requireRef(in.Ref)
	if err != nil {
		return domain.Proposal{}, err
	}
	proposal, err := d.Proposals.ProposalByRef(ctx, ref)
	if err != nil {
		return domain.Proposal{}, err
	}
	if !proposal.Decidable() {
		return proposal, fmt.Errorf("%w: proposal is %s, and a decision is final", domain.ErrConflict, proposal.Status)
	}

	next := domain.StatusRejected
	if in.Approve {
		next = domain.StatusApproved
		if proposal.Status != domain.StatusVerified {
			return proposal, fmt.Errorf("%w: proposal is %s, and only a verified proposal can be approved",
				domain.ErrConflict, proposal.Status)
		}
	}
	if !proposal.CanTransition(next) {
		return proposal, fmt.Errorf("%w: %s cannot become %s", domain.ErrConflict, proposal.Status, next)
	}
	// Separation of duties: a rejection may be the proposer withdrawing their own idea, but an
	// approval from the same token is a rubber stamp unless that token is explicitly an admin.
	if in.Approve && proposal.ProposedBy == in.Actor && !in.ActorIsAdmin {
		return proposal, fmt.Errorf("%w: %q proposed this change, so it cannot also approve it", domain.ErrForbidden, in.Actor)
	}

	now := time.Now().UTC()
	proposal.Status = next
	proposal.DecidedBy = in.Actor
	proposal.DecidedAt = &now
	proposal.DecidedNote = in.Note
	proposal.UpdatedAt = now
	stored, err := d.Proposals.UpdateProposal(ctx, proposal)
	if err != nil {
		return proposal, err
	}

	kind := constants.JournalRejected
	if in.Approve {
		kind = constants.JournalApproved
	}
	if err := d.record(ctx, stored, kind, map[string]string{"by": in.Actor, "note": in.Note}); err != nil {
		return stored, err
	}
	if in.Approve && proposal.ProposedBy == in.Actor {
		// The exception is recorded as its own line: an approval that skipped the separation is a
		// thing a reviewer must be able to find later.
		if err := d.record(ctx, stored, constants.JournalOverride, map[string]string{
			"by": in.Actor, "reason": "admin approved a proposal it also made",
		}); err != nil {
			return stored, err
		}
	}
	return stored, nil
}
