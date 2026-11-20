package proposal

import (
	"context"
	"fmt"
	"noxis/internal/common/ref"
	"noxis/internal/common/sqltext"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"time"
)

// Create turns one statement and its rollback into a proposal, then rehearses it immediately: a
// proposal that has not been rehearsed is an idea, and the queue is for evidence.
type Create struct{ Deps }

// NewCreate builds the use case.
func NewCreate(d Deps) *Create { return &Create{Deps: d} }

// CreateInput is one proposal as the caller writes it.
type CreateInput struct {
	TargetRef string
	Title     string
	Query     string
	Rollback  string
	Note      string
	Tier      string
	Actor     string
}

// CreateOutput is the proposal as it now stands and the evidence behind it.
type CreateOutput struct {
	Proposal  domain.Proposal
	Rehearsal domain.Rehearsal
}

// Run classifies, stores, rehearses and journals.
func (c *Create) Run(ctx context.Context, in CreateInput) (out0 CreateOutput, err error) {
	ctx, end := telemetry.Span(ctx, "proposal.Create.Run")
	defer func() { end(err) }()

	targetRef, err := requireRef(in.TargetRef)
	if err != nil {
		return CreateOutput{}, err
	}
	tier := domain.Tier(in.Tier)
	if in.Tier == "" {
		tier = domain.TierShadow
	}
	if !domain.IsTier(string(tier)) {
		return CreateOutput{}, fmt.Errorf("%w: tier %q is not one of static, shadow, rehearsal", domain.ErrInvalid, in.Tier)
	}

	kind, err := kindOf(in.Query)
	if err != nil {
		return CreateOutput{}, err
	}

	now := time.Now().UTC()
	proposal := domain.Proposal{
		Ref:        ref.New(),
		TargetRef:  targetRef,
		Title:      strings.TrimSpace(in.Title),
		Kind:       kind,
		Query:      strings.TrimSpace(in.Query),
		Rollback:   strings.TrimSpace(in.Rollback),
		Note:       strings.TrimSpace(in.Note),
		Status:     domain.StatusDraft,
		Tier:       tier,
		ProposedBy: in.Actor,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := proposal.Validate(); err != nil {
		return CreateOutput{}, err
	}

	stored, err := c.Proposals.CreateProposal(ctx, proposal)
	if err != nil {
		return CreateOutput{}, err
	}
	if err := c.record(ctx, stored, constants.JournalProposed, map[string]string{
		"title": stored.Title,
		"kind":  string(stored.Kind),
		"tier":  string(stored.Tier),
	}); err != nil {
		return CreateOutput{}, err
	}
	rehearsed, rehearsal, err := c.rehearse(ctx, stored)
	if err != nil {
		return CreateOutput{Proposal: rehearsed, Rehearsal: rehearsal}, err
	}
	return CreateOutput{Proposal: rehearsed, Rehearsal: rehearsal}, nil
}

// kindOf maps a statement onto what it changes, and refuses anything that changes nothing: a read
// is not a proposal, it is a read.
func kindOf(query string) (domain.Kind, error) {
	statement, err := sqltext.Classify(query)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	switch statement.Kind {
	case sqltext.DDL:
		return domain.KindDDL, nil
	case sqltext.Insert, sqltext.Update, sqltext.Delete:
		return domain.KindDML, nil
	case sqltext.Select, sqltext.Explain, sqltext.Show:
		return "", fmt.Errorf("%w: a %s changes nothing, so there is nothing to approve", domain.ErrInvalid, statement.Kind)
	}
	return "", fmt.Errorf("%w: %q is not a change this build knows how to rehearse", domain.ErrInvalid, statement.Kind)
}
