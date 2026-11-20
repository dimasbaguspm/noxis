package proposal

import (
	"context"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"strings"
	"time"
)

// The doubles are handwritten and live with the tests that use them: a generated mock would hide
// which call a test depends on, and this package only ever needs these six.

type fakeProposals struct {
	stored     map[string]domain.Proposal
	list       []domain.Proposal
	createErr  error
	readErr    error
	listErr    error
	updateErr  error
	claimErr   error
	progress   []domain.Progress
	claimFrom  domain.Status
	updateCall int
}

func newFakeProposals(seed ...domain.Proposal) *fakeProposals {
	fake := &fakeProposals{stored: map[string]domain.Proposal{}}
	for _, proposal := range seed {
		fake.stored[proposal.Ref] = proposal
	}
	return fake
}

func (f *fakeProposals) CreateProposal(_ context.Context, proposal domain.Proposal) (domain.Proposal, error) {
	if f.createErr != nil {
		return domain.Proposal{}, f.createErr
	}
	f.stored[proposal.Ref] = proposal
	return proposal, nil
}

func (f *fakeProposals) ProposalByRef(_ context.Context, ref string) (domain.Proposal, error) {
	if f.readErr != nil {
		return domain.Proposal{}, f.readErr
	}
	proposal, ok := f.stored[ref]
	if !ok {
		return domain.Proposal{}, domain.ErrNotFound
	}
	return proposal, nil
}

func (f *fakeProposals) ListProposals(_ context.Context, filter ports.ProposalFilter) ([]domain.Proposal, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.list = f.list[:0]
	for _, proposal := range f.stored {
		f.list = append(f.list, proposal)
	}
	if len(f.list) > filter.Limit {
		return f.list[:filter.Limit], nil
	}
	return f.list, nil
}

func (f *fakeProposals) UpdateProposal(_ context.Context, proposal domain.Proposal) (domain.Proposal, error) {
	f.updateCall++
	if f.updateErr != nil {
		return domain.Proposal{}, f.updateErr
	}
	f.stored[proposal.Ref] = proposal
	return proposal, nil
}

func (f *fakeProposals) ClaimProposal(_ context.Context, ref string, from, to domain.Status) (domain.Proposal, error) {
	if f.claimErr != nil {
		return domain.Proposal{}, f.claimErr
	}
	proposal, ok := f.stored[ref]
	if !ok {
		return domain.Proposal{}, domain.ErrNotFound
	}
	if proposal.Status != from {
		return proposal, domain.ErrConflict
	}
	f.claimFrom = from
	proposal.Status = to
	f.stored[ref] = proposal
	return proposal, nil
}

func (f *fakeProposals) SetProgress(_ context.Context, _ string, progress domain.Progress) error {
	f.progress = append(f.progress, progress)
	return nil
}

type fakeRehearsals struct {
	latest  domain.Rehearsal
	err     error
	created []domain.Rehearsal
}

func (f *fakeRehearsals) CreateRehearsal(_ context.Context, rehearsal domain.Rehearsal) (domain.Rehearsal, error) {
	f.created = append(f.created, rehearsal)
	return rehearsal, nil
}

func (f *fakeRehearsals) RehearsalByRef(_ context.Context, ref string) (domain.Rehearsal, error) {
	if f.err != nil {
		return domain.Rehearsal{}, f.err
	}
	if f.latest.Ref != ref {
		return domain.Rehearsal{}, domain.ErrNotFound
	}
	return f.latest, nil
}

func (f *fakeRehearsals) LatestRehearsal(_ context.Context, _ string) (domain.Rehearsal, error) {
	if f.err != nil {
		return domain.Rehearsal{}, f.err
	}
	return f.latest, nil
}

type fakeTargets struct {
	target domain.Target
	err    error
}

func (*fakeTargets) Create(_ context.Context, target domain.Target) (domain.Target, error) {
	return target, nil
}

func (f *fakeTargets) ByRef(_ context.Context, _ string) (domain.Target, error) {
	if f.err != nil {
		return domain.Target{}, f.err
	}
	return f.target, nil
}

func (*fakeTargets) List(_ context.Context, _ ports.TargetFilter) ([]domain.Target, error) {
	return nil, nil
}

func (*fakeTargets) Update(_ context.Context, target domain.Target) (domain.Target, error) {
	return target, nil
}

type fakeJournal struct {
	entries []domain.JournalEntry
	err     error
}

func (f *fakeJournal) Append(_ context.Context, entries ...domain.JournalEntry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, entries...)
	return nil
}

func (f *fakeJournal) Entries(_ context.Context, subjectRef string, _ int) ([]domain.JournalEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	var found []domain.JournalEntry
	for _, entry := range f.entries {
		if entry.SubjectRef == subjectRef {
			found = append(found, entry)
		}
	}
	return found, nil
}

// fakeSealer seals nothing and says so by prefix: the tests assert the plaintext never reaches a
// stored row, and this is the shape that makes that visible.
type fakeSealer struct {
	enabled bool
	openErr error
}

func (f *fakeSealer) Enabled() bool { return f.enabled }

func (*fakeSealer) Seal(_ string, plain []byte) (string, error) {
	return "sealed:" + string(plain), nil
}

func (f *fakeSealer) Open(_ string, sealed string) ([]byte, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
}

// fakeDriver answers what each test says it should. A driver that guesses is a driver whose failure
// path can never be reached.
type fakeDriver struct {
	caps        domain.Capabilities
	probeErr    error
	rows        domain.Rows
	queryErr    error
	databases   []string
	tables      []domain.Table
	table       domain.TableDetail
	rehearsal   domain.Rehearsal
	rehearseErr error
	plan        domain.Plan
	planErr     error
	receipt     domain.Receipt
	applyErr    error
	applied     []domain.Change
}

func (f *fakeDriver) Probe(_ context.Context, _ domain.Target, _ string) (domain.Capabilities, error) {
	return f.caps, f.probeErr
}

func (f *fakeDriver) Query(_ context.Context, _ domain.Target, _, _ string, _ int) (domain.Rows, error) {
	return f.rows, f.queryErr
}

func (f *fakeDriver) Databases(_ context.Context, _ domain.Target, _ string) ([]string, error) {
	return f.databases, nil
}

func (f *fakeDriver) Tables(_ context.Context, _ domain.Target, _, _ string) ([]domain.Table, error) {
	return f.tables, nil
}

func (f *fakeDriver) Table(_ context.Context, _ domain.Target, _, _, _ string) (domain.TableDetail, error) {
	return f.table, nil
}

func (f *fakeDriver) Rehearse(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Rehearsal, error) {
	return f.rehearsal, f.rehearseErr
}

func (f *fakeDriver) Plan(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Plan, error) {
	return f.plan, f.planErr
}

func (f *fakeDriver) Apply(_ context.Context, _ domain.Target, _ string, change domain.Change,
	_ func(domain.Progress),
) (domain.Receipt, error) {
	f.applied = append(f.applied, change)
	return f.receipt, f.applyErr
}

// fixture is one built set of use cases over one target and one proposal.
type fixture struct {
	deps      Deps
	target    domain.Target
	proposal  domain.Proposal
	proposals *fakeProposals
	rehearsal *fakeRehearsals
	targets   *fakeTargets
	journal   *fakeJournal
	sealer    *fakeSealer
	driver    *fakeDriver
}

func newFixture(t interface{ Helper() }, status domain.Status) *fixture {
	t.Helper()
	target := domain.Target{
		Ref: "0192f0c0-0000-7000-8000-000000000001", Name: "prod", Driver: "mysql",
		Host: "db", Port: 3306, Database: "app", Username: "noxis", Secret: "sealed:pw",
		TLSMode: "preferred", CreatedBy: "ops",
	}
	proposal := domain.Proposal{
		Ref: "0192f0c0-0000-7000-8000-000000000002", TargetRef: target.Ref,
		Title: "add a column", Kind: domain.KindDDL, Query: "ALTER TABLE users ADD COLUMN note TEXT",
		Tier: domain.TierShadow, Status: status, ProposedBy: "agent",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}

	f := &fixture{
		target:    target,
		proposal:  proposal,
		proposals: newFakeProposals(proposal),
		rehearsal: &fakeRehearsals{latest: domain.Rehearsal{Status: domain.RehearsalVerified}},
		targets:   &fakeTargets{target: target},
		journal:   &fakeJournal{},
		sealer:    &fakeSealer{enabled: true},
		driver:    &fakeDriver{},
	}
	f.deps = Deps{
		Proposals: f.proposals, Rehearsals: f.rehearsal, Targets: f.targets,
		Journal: f.journal, Sealer: f.sealer, Driver: f.driver, Timeout: time.Second,
	}
	return f
}
