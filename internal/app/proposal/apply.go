package proposal

import (
	"context"
	"fmt"
	"noxis/internal/app/password"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
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

// ApplyOutput is what one attempt answers: the proposal as it stands, and whether the statement is
// still running behind it. `Started` is the difference between "done, read the receipt" and "202,
// poll the proposal".
type ApplyOutput struct {
	Proposal domain.Proposal
	Started  bool
}

// Run claims the proposal, decides whether the statement can answer in the call, and applies it. A
// failure is stored on the proposal: a silent retry is how the same change lands twice.
func (a *Apply) Run(ctx context.Context, in ApplyInput) (out0 ApplyOutput, err error) {
	ctx, end := telemetry.Span(ctx, "proposal.Apply.Run")
	defer func() { end(err) }()

	ref, err := requireRef(in.Ref)
	if err != nil {
		return ApplyOutput{}, err
	}
	proposal, err := a.Proposals.ProposalByRef(ctx, ref)
	if err != nil {
		return ApplyOutput{}, err
	}
	if !proposal.Applyable() {
		return ApplyOutput{Proposal: proposal}, fmt.Errorf("%w: proposal is %s, and only an approved proposal can be applied",
			domain.ErrConflict, proposal.Status)
	}
	if proposal.Irreversible && !in.ConfirmDestructive {
		return ApplyOutput{Proposal: proposal}, fmt.Errorf("%w: this change is irreversible; send confirm_destructive to run it anyway",
			domain.ErrInvalid)
	}

	// The claim is the lock. If somebody else already applied it, this returns their state and the
	// statement below never runs.
	claimed, err := a.Proposals.ClaimProposal(ctx, proposal.Ref, domain.StatusApproved, domain.StatusApplying)
	if err != nil {
		return ApplyOutput{Proposal: claimed}, err
	}

	target, err := a.Targets.ByRef(ctx, claimed.TargetRef)
	if err != nil {
		return ApplyOutput{Proposal: claimed}, err
	}
	plain, err := password.Open(a.Sealer, target)
	if err != nil {
		return ApplyOutput{Proposal: claimed}, err
	}
	change := domain.Change{Query: claimed.Query, Rollback: claimed.Rollback, Tier: claimed.Tier}

	// How big is this? Sizing the change is what decides whether the answer waits. A failure to size
	// it is not a reason to refuse it: the statement is still correct, only unmeasured.
	plan, err := a.Driver.Plan(ctx, target, plain, change)
	if err != nil {
		plan = domain.Plan{}
	}
	started := time.Now().UTC()
	claimed.StartedAt = &started
	claimed.RowsEstimated = plan.RowsEstimated
	claimed.Progress = domain.UnknownProgress
	claimed.UpdatedAt = started
	if claimed, err = a.Proposals.UpdateProposal(ctx, claimed); err != nil {
		return ApplyOutput{Proposal: claimed}, err
	}

	if plan.RowsEstimated >= constants.ApplyBackgroundRows {
		// The claim already happened, so this runs once even if two operators ask at the same time.
		// The work outlives the request on purpose: the caller is gone by design, and cancelling a
		// table copy with its poll would make the 202 a lie.
		detached := context.WithoutCancel(ctx)
		go func() { _, _ = a.finish(detached, claimed, target, plain, change, in.Actor) }()
		return ApplyOutput{Proposal: claimed, Started: true}, nil
	}

	finished, err := a.finish(ctx, claimed, target, plain, change, in.Actor)
	return ApplyOutput{Proposal: finished}, err
}

// finish runs the statement, records what the engine said while it ran, and writes the receipt. It
// is the same code whether the caller is still on the line or reading the proposal tomorrow.
func (a *Apply) finish(ctx context.Context, proposal domain.Proposal, target domain.Target, plain string,
	change domain.Change, actor string,
) (domain.Proposal, error) {
	bounded, cancel := context.WithTimeout(ctx, a.Timeout)
	defer cancel()

	report := func(progress domain.Progress) {
		// Progress is an observation: a write that fails loses one datapoint, not the change.
		_ = a.Proposals.SetProgress(bounded, proposal.Ref, progress)
	}
	receipt, applyErr := a.Driver.Apply(bounded, target, plain, change, report)

	now := time.Now().UTC()
	proposal.UpdatedAt = now
	proposal.Progress = domain.UnknownProgress
	proposal.RowsAffected = receipt.RowsAffected
	if receipt.Algorithm != "" {
		proposal.Algorithm = receipt.Algorithm
	}

	if applyErr != nil {
		proposal.Status = domain.StatusFailed
		proposal.Failure = applyErr.Error()
		stored, updateErr := a.Proposals.UpdateProposal(bounded, proposal)
		if updateErr != nil {
			return proposal, updateErr
		}
		if err := a.record(bounded, stored, constants.JournalApplyFailed, map[string]string{
			"by": actor, "error": truncate(applyErr.Error()),
		}); err != nil {
			return stored, err
		}
		return stored, applyErr
	}

	proposal.Status = domain.StatusApplied
	proposal.AppliedBy = actor
	proposal.AppliedAt = &now
	proposal.Failure = ""
	stored, err := a.Proposals.UpdateProposal(bounded, proposal)
	if err != nil {
		return proposal, err
	}
	detail := map[string]string{"by": actor, "rows_affected": fmt.Sprintf("%d", receipt.RowsAffected)}
	if proposal.Algorithm != "" {
		detail["algorithm"] = string(proposal.Algorithm)
	}
	if err := a.record(bounded, stored, constants.JournalApplied, detail); err != nil {
		return stored, err
	}
	return stored, nil
}
