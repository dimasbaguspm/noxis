package handlers

import (
	"context"
	"noxis/internal/domain"
	"noxis/internal/mocks"
	"noxis/internal/ports"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
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

// ByKey scans for a stored key: the fake keeps rows by ref, and this is the one lookup that does not
// start from one. It is what makes the retry test over HTTP possible.
func (s *store) ByKey(_ context.Context, key string) (domain.Target, error) {
	for _, target := range s.targets {
		if target.IdempotencyKey == key {
			return target, nil
		}
	}
	return domain.Target{}, domain.ErrNotFound
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

// ProposalByKey is the same scan for a proposal: a retried propose finds what the first one made.
func (s *store) ProposalByKey(_ context.Context, key string) (domain.Proposal, error) {
	for _, proposal := range s.proposals {
		if proposal.IdempotencyKey == key {
			return proposal, nil
		}
	}
	return domain.Proposal{}, domain.ErrNotFound
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

func (*store) SetProgress(_ context.Context, _ string, _ domain.Progress) error { return nil }

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

// newSealer is the generated mock, wired to the fake's own rules: a password is "sealed" by prefix,
// which is enough to prove nothing a handler returns carries it.
func newSealer(t *testing.T) *mocks.MockSealer {
	sealer := mocks.NewMockSealer(gomock.NewController(t))
	sealer.EXPECT().Enabled().Return(true).AnyTimes()
	sealer.EXPECT().Seal(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ string, plain []byte) (string, error) {
			return "sealed:" + string(plain), nil
		}).AnyTimes()
	sealer.EXPECT().Open(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_, sealed string) ([]byte, error) {
			return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
		}).AnyTimes()
	return sealer
}

// newDriver answers every engine call from the fake's fields, read at call time, so a test can still
// set what the engine says after the handlers were built.
func newDriver(t *testing.T, s *store) *mocks.MockDriver {
	driver := mocks.NewMockDriver(gomock.NewController(t))
	driver.EXPECT().Probe(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string) (domain.Capabilities, error) {
			return s.driverCaps, s.driverErr
		}).AnyTimes()
	driver.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, string, int) (domain.Rows, error) {
			return s.driverRows, s.driverErr
		}).AnyTimes()
	driver.EXPECT().Databases(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string) ([]string, error) {
			return nil, s.driverErr
		}).AnyTimes()
	driver.EXPECT().Tables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, string) ([]domain.Table, error) {
			return nil, s.driverErr
		}).AnyTimes()
	driver.EXPECT().Table(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, string, string) (domain.TableDetail, error) {
			return domain.TableDetail{}, s.driverErr
		}).AnyTimes()
	driver.EXPECT().Rehearse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, domain.Change) (domain.Rehearsal, error) {
			return s.rehearsal, s.rehearseErr
		}).AnyTimes()
	driver.EXPECT().Plan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, domain.Change) (domain.Plan, error) {
			return s.plan, nil
		}).AnyTimes()
	driver.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.Target, string, domain.Change, func(domain.Progress)) (domain.Receipt, error) {
			return s.receipt, s.applyErr
		}).AnyTimes()
	return driver
}
