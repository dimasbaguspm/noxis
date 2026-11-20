package proposal

import (
	"context"
	"fmt"
	"time"

	"noxis/internal/app/password"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

// Apply runs an approved proposal for real. The claim happens before the statement does, so a crash
// leaves `applying` — indeterminate and unmistakable — instead of a change nobody knows about.
type Apply struct{ Deps }

// NewApply builds the use case.
func NewApply(d Deps) *Apply { return &Apply{Deps: d} }

// ApplyInput names the proposal and acknowledges a change that cannot be undone.
type ApplyInput struct {
	Ref                string
	Actor              string
	ConfirmDestructive bool
}

// Run claims, applies and records the receipt. A failure is stored on the proposal: a silent retry
// is how the same change lands twice.
func (a *Apply) Run(ctx context.Context, in ApplyInput) (domain.Proposal, error) {
	ref, err := requireRef(in.Ref)
	if err != nil {
		return domain.Proposal{}, err
	}
	proposal, err := a.Proposals.ProposalByRef(ctx, ref)
	if err != nil {
		return domain.Proposal{}, err
	}
	if !proposal.Applyable() {
		return proposal, fmt.Errorf("%w: proposal is %s, and only an approved proposal can be applied",
			domain.ErrConflict, proposal.Status)
	}
	if proposal.Irreversible && !in.ConfirmDestructive {
		return proposal, fmt.Errorf("%w: this change is irreversible; send confirm_destructive to run it anyway",
			domain.ErrInvalid)
	}

	// The claim is the lock. If somebody else already applied it, this returns their state and the
	// statement below never runs.
	claimed, err := a.Proposals.ClaimProposal(ctx, proposal.Ref, domain.StatusApproved, domain.StatusApplying)
	if err != nil {
		return claimed, err
	}

	target, err := a.Targets.ByRef(ctx, claimed.TargetRef)
	if err != nil {
		return claimed, err
	}
	plain, _, err := password.Open(a.Sealer, target)
	if err != nil {
		return claimed, err
	}

	bounded, cancel := context.WithTimeout(ctx, a.Timeout)
	defer cancel()
	applyErr := a.Driver.Apply(bounded, target, plain, domain.Change{
		Query:    claimed.Query,
		Rollback: claimed.Rollback,
		Tier:     claimed.Tier,
	})

	now := time.Now().UTC()
	claimed.UpdatedAt = now
	if applyErr != nil {
		claimed.Status = domain.StatusFailed
		claimed.Failure = applyErr.Error()
		stored, updateErr := a.Proposals.UpdateProposal(ctx, claimed)
		if updateErr != nil {
			return claimed, updateErr
		}
		if err := a.record(ctx, stored, constants.JournalApplyFailed, map[string]string{
			"by": in.Actor, "error": truncate(applyErr.Error()),
		}); err != nil {
			return stored, err
		}
		return stored, applyErr
	}

	claimed.Status = domain.StatusApplied
	claimed.AppliedBy = in.Actor
	claimed.AppliedAt = &now
	claimed.Failure = ""
	stored, err := a.Proposals.UpdateProposal(ctx, claimed)
	if err != nil {
		return claimed, err
	}
	if err := a.record(ctx, stored, constants.JournalApplied, map[string]string{"by": in.Actor}); err != nil {
		return stored, err
	}
	return stored, nil
}
