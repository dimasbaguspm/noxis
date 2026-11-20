package handlers

import (
	"context"
	"strings"
	"time"

	"noxis/internal/domain"
	"noxis/internal/ports"
)

// store is one double for every port the use cases hold. The handlers only ever see the four use
// case groups, so a single memory-shaped store keeps the test readable.
type store struct {
	targets    map[string]domain.Target
	proposals  map[string]domain.Proposal
	rehearsals map[string]domain.Rehearsal
	journal    []domain.JournalEntry

	driverCaps  domain.Capabilities
	driverRows  domain.Rows
	driverErr   error
	plan        domain.Plan
	receipt     domain.Receipt
	applyErr    error
	rehearsal   domain.Rehearsal
	rehearseErr error
}

func newStore() *store {
	return &store{
		targets:    map[string]domain.Target{},
		proposals:  map[string]domain.Proposal{},
		rehearsals: map[string]domain.Rehearsal{},
	}
}

// ---- targets ----

func (s *store) Create(_ context.Context, target domain.Target) (domain.Target, error) {
	s.targets[target.Ref] = target
	return target, nil
}

func (s *store) ByRef(_ context.Context, ref string) (domain.Target, error) {
	target, ok := s.targets[ref]
	if !ok {
		return domain.Target{}, domain.ErrNotFound
	}
	return target, nil
}

func (s *store) List(_ context.Context, filter ports.TargetFilter) ([]domain.Target, error) {
	found := make([]domain.Target, 0, len(s.targets))
	for _, target := range s.targets {
		found = append(found, target)
	}
	if len(found) > filter.Limit {
		return found[:filter.Limit], nil
	}
	return found, nil
}

func (s *store) Update(_ context.Context, target domain.Target) (domain.Target, error) {
	if _, ok := s.targets[target.Ref]; !ok {
		return domain.Target{}, domain.ErrNotFound
	}
	s.targets[target.Ref] = target
	return target, nil
}

// ---- journal ----

func (s *store) Append(_ context.Context, entries ...domain.JournalEntry) error {
	s.journal = append(s.journal, entries...)
	return nil
}

func (s *store) Entries(_ context.Context, subjectRef string, _ int) ([]domain.JournalEntry, error) {
	var found []domain.JournalEntry
	for _, entry := range s.journal {
		if entry.SubjectRef == subjectRef {
			found = append(found, entry)
		}
	}
	return found, nil
}

// ---- sealer ----

type testSealer struct{}

func (testSealer) Enabled() bool { return true }

func (testSealer) Seal(_ string, plain []byte) (string, error) { return "sealed:" + string(plain), nil }

func (testSealer) Open(_ string, sealed string) ([]byte, bool, error) {
	return []byte(strings.TrimPrefix(sealed, "sealed:")), false, nil
}

// ---- driver ----

type testDriver struct{ s *store }

func (d testDriver) Probe(_ context.Context, _ domain.Target, _ string) (domain.Capabilities, error) {
	return d.s.driverCaps, d.s.driverErr
}

func (d testDriver) Query(_ context.Context, _ domain.Target, _, _ string, _ int) (domain.Rows, error) {
	return d.s.driverRows, d.s.driverErr
}

func (d testDriver) Databases(_ context.Context, _ domain.Target, _ string) ([]string, error) {
	return nil, d.s.driverErr
}

func (d testDriver) Tables(_ context.Context, _ domain.Target, _, _ string) ([]domain.Table, error) {
	return nil, d.s.driverErr
}

func (d testDriver) Table(_ context.Context, _ domain.Target, _, _, _ string) (domain.TableDetail, error) {
	return domain.TableDetail{}, d.s.driverErr
}

func (d testDriver) Rehearse(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Rehearsal, error) {
	return d.s.rehearsal, d.s.rehearseErr
}

func (d testDriver) Plan(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Plan, error) {
	return d.s.plan, nil
}

func (d testDriver) Apply(_ context.Context, _ domain.Target, _ string, _ domain.Change,
	_ func(domain.Progress)) (domain.Receipt, error) {
	return d.s.receipt, d.s.applyErr
}

// ---- proposals ----

func (s *store) CreateProposal(_ context.Context, proposal domain.Proposal) (domain.Proposal, error) {
	s.proposals[proposal.Ref] = proposal
	return proposal, nil
}

func (s *store) ProposalByRef(_ context.Context, ref string) (domain.Proposal, error) {
	proposal, ok := s.proposals[ref]
	if !ok {
		return domain.Proposal{}, domain.ErrNotFound
	}
	return proposal, nil
}

func (s *store) ListProposals(_ context.Context, filter ports.ProposalFilter) ([]domain.Proposal, error) {
	found := make([]domain.Proposal, 0, len(s.proposals))
	for _, proposal := range s.proposals {
		found = append(found, proposal)
	}
	if len(found) > filter.Limit {
		return found[:filter.Limit], nil
	}
	return found, nil
}

func (s *store) UpdateProposal(_ context.Context, proposal domain.Proposal) (domain.Proposal, error) {
	s.proposals[proposal.Ref] = proposal
	return proposal, nil
}

func (s *store) ClaimProposal(_ context.Context, ref string, from, to domain.Status) (domain.Proposal, error) {
	proposal, ok := s.proposals[ref]
	if !ok {
		return domain.Proposal{}, domain.ErrNotFound
	}
	if proposal.Status != from {
		return proposal, domain.ErrConflict
	}
	proposal.Status = to
	s.proposals[ref] = proposal
	return proposal, nil
}

func (s *store) SetProgress(_ context.Context, _ string, _ domain.Progress) error { return nil }

// ---- rehearsals ----

func (s *store) CreateRehearsal(_ context.Context, rehearsal domain.Rehearsal) (domain.Rehearsal, error) {
	s.rehearsals[rehearsal.Ref] = rehearsal
	return rehearsal, nil
}

func (s *store) RehearsalByRef(_ context.Context, ref string) (domain.Rehearsal, error) {
	rehearsal, ok := s.rehearsals[ref]
	if !ok {
		return domain.Rehearsal{}, domain.ErrNotFound
	}
	return rehearsal, nil
}

func (s *store) LatestRehearsal(_ context.Context, proposalRef string) (domain.Rehearsal, error) {
	for _, rehearsal := range s.rehearsals {
		if rehearsal.ProposalRef == proposalRef {
			return rehearsal, nil
		}
	}
	return domain.Rehearsal{}, domain.ErrNotFound
}

// seedTarget stores one target that passes Validate and returns it.
func (s *store) seedTarget(ref string) domain.Target {
	now := time.Now().UTC()
	target := domain.Target{
		Ref: ref, Name: "prod", Driver: "mysql", Host: "db.internal", Port: 3306,
		Database: "app", Username: "noxis", Secret: "sealed:pw", TLSMode: "preferred",
		CreatedBy: "ops", CreatedAt: now, UpdatedAt: now,
	}
	s.targets[ref] = target
	return target
}

// seedProposal stores one proposal in the given status and returns it.
func (s *store) seedProposal(ref, targetRef string, status domain.Status) domain.Proposal {
	now := time.Now().UTC()
	proposal := domain.Proposal{
		Ref: ref, TargetRef: targetRef, Title: "add a column", Kind: domain.KindDDL,
		Query: "ALTER TABLE users ADD COLUMN note TEXT", Tier: domain.TierShadow,
		Status: status, ProposedBy: "agent", CreatedAt: now, UpdatedAt: now,
	}
	s.proposals[ref] = proposal
	return proposal
}
