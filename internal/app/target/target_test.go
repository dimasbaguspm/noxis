package target

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"noxis/internal/domain"
	"noxis/internal/ports"
)

// ---- doubles (handwritten, in the package that uses them) ----

type fakeTargets struct {
	target  domain.Target
	stored  []domain.Target
	err     error
	updateErr error
}

func (f *fakeTargets) Create(_ context.Context, target domain.Target) (domain.Target, error) {
	if f.err != nil {
		return domain.Target{}, f.err
	}
	f.stored = append(f.stored, target)
	return target, nil
}

func (f *fakeTargets) ByRef(_ context.Context, _ string) (domain.Target, error) {
	if f.err != nil {
		return domain.Target{}, f.err
	}
	return f.target, nil
}

func (f *fakeTargets) List(_ context.Context, filter ports.TargetFilter) ([]domain.Target, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.stored) > filter.Limit {
		return f.stored[:filter.Limit], nil
	}
	return f.stored, nil
}

func (f *fakeTargets) Update(_ context.Context, target domain.Target) (domain.Target, error) {
	if f.updateErr != nil {
		return domain.Target{}, f.updateErr
	}
	f.target = target
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

func (f *fakeJournal) Entries(_ context.Context, _ string, _ int) ([]domain.JournalEntry, error) {
	return f.entries, nil
}

type fakeSealer struct {
	enabled bool
	openErr error
	sealErr error
	stale   bool
}

func (f *fakeSealer) Enabled() bool { return f.enabled }

func (f *fakeSealer) Seal(_ string, plain []byte) (string, error) {
	if f.sealErr != nil {
		return "", f.sealErr
	}
	return "sealed:" + string(plain), nil
}

func (f *fakeSealer) Open(_ string, sealed string) ([]byte, bool, error) {
	if f.openErr != nil {
		return nil, false, f.openErr
	}
	return []byte(strings.TrimPrefix(sealed, "sealed:")), f.stale, nil
}

type fakeDriver struct {
	caps     domain.Capabilities
	probeErr error
}

func (f *fakeDriver) Probe(_ context.Context, _ domain.Target, _ string) (domain.Capabilities, error) {
	return f.caps, f.probeErr
}

func (f *fakeDriver) Query(_ context.Context, _ domain.Target, _, _ string, _ int) (domain.Rows, error) {
	return domain.Rows{}, nil
}

func (f *fakeDriver) Databases(_ context.Context, _ domain.Target, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeDriver) Tables(_ context.Context, _ domain.Target, _, _ string) ([]domain.Table, error) {
	return nil, nil
}

func (f *fakeDriver) Table(_ context.Context, _ domain.Target, _, _, _ string) (domain.TableDetail, error) {
	return domain.TableDetail{}, nil
}

func (f *fakeDriver) Rehearse(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Rehearsal, error) {
	return domain.Rehearsal{}, nil
}

func (f *fakeDriver) Plan(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Plan, error) {
	return domain.Plan{}, nil
}

func (f *fakeDriver) Apply(_ context.Context, _ domain.Target, _ string, _ domain.Change,
	_ func(domain.Progress)) (domain.Receipt, error) {
	return domain.Receipt{}, nil
}

// fixture is one target that passes Validate, and the doubles that carry it.
type fixture struct {
	deps    Deps
	target  domain.Target
	targets *fakeTargets
	journal *fakeJournal
	sealer  *fakeSealer
	driver  *fakeDriver
}

func newFixture() *fixture {
	target := domain.Target{
		Ref: "0192f0c0-0000-7000-8000-000000000001", Name: "prod", Driver: "mysql",
		Host: "db.internal", Port: 3306, Database: "app", Username: "noxis",
		Secret: "sealed:pw", TLSMode: "preferred", CreatedBy: "ops",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	f := &fixture{
		target:  target,
		targets: &fakeTargets{target: target},
		journal: &fakeJournal{},
		sealer:  &fakeSealer{enabled: true},
		driver:  &fakeDriver{},
	}
	f.deps = Deps{Targets: f.targets, Journal: f.journal, Sealer: f.sealer, Driver: f.driver}
	return f
}

// ---- create ----

// A password that arrives with a target is stored sealed, and the value handed back never carries
// it: the caller reads what the state file would give them, not what they typed.
func TestCreateSealsThePassword(t *testing.T) {
	f := newFixture()

	created, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		Name: "staging", Driver: "mysql", Host: "db", Database: "app", Username: "noxis",
		Password: "hunter2", Actor: "ops",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Secret != "sealed:hunter2" {
		t.Fatalf("secret = %q, want the sealed envelope", created.Secret)
	}
	if created.Port != 3306 || created.TLSMode != "preferred" {
		t.Fatalf("defaults = %d/%s", created.Port, created.TLSMode)
	}
	if len(f.journal.entries) != 1 || f.journal.entries[0].Kind != "created" {
		t.Fatalf("journal = %+v, want one `created` line", f.journal.entries)
	}
}

// A password with no key to seal it with is refused: storing it in the clear is not a fallback.
func TestCreateRefusesAPasswordWithoutAKey(t *testing.T) {
	f := newFixture()
	f.sealer.enabled = false

	_, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		Name: "staging", Driver: "mysql", Host: "db", Database: "app", Username: "noxis",
		Password: "hunter2", Actor: "ops",
	})
	if !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("Create with no key: %v, want ErrUnsupported", err)
	}
	if len(f.targets.stored) != 0 {
		t.Fatal("the target was stored without its password")
	}
}

func TestCreateRefusals(t *testing.T) {
	f := newFixture()
	base := CreateInput{Name: "staging", Driver: "mysql", Host: "db", Database: "app", Username: "noxis", Actor: "ops"}

	for _, tc := range []struct {
		name   string
		change func(*CreateInput)
	}{
		{"bad name", func(in *CreateInput) { in.Name = "Staging!" }},
		{"bad driver", func(in *CreateInput) { in.Driver = "postgres" }},
		{"no host", func(in *CreateInput) { in.Host = "" }},
		{"no actor", func(in *CreateInput) { in.Actor = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.change(&in)
			if _, err := NewCreate(f.deps).Run(context.Background(), in); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Create: %v, want ErrInvalid", err)
			}
		})
	}
}

// ---- read one, list ----

func TestGetRequiresAReference(t *testing.T) {
	f := newFixture()
	if _, err := NewGet(f.deps).Run(context.Background(), "prod"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Get: %v, want ErrInvalid", err)
	}
	if _, err := NewGet(f.deps).Run(context.Background(), f.target.Ref); err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestListPagesAndFilters(t *testing.T) {
	f := newFixture()
	f.targets.stored = []domain.Target{f.target, f.target, f.target}

	out, err := NewList(f.deps).Run(context.Background(), ListInput{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out.Targets) != 2 || out.NextCursor == "" {
		t.Fatalf("page = %d rows, cursor %q", len(out.Targets), out.NextCursor)
	}

	if _, err := NewList(f.deps).Run(context.Background(), ListInput{Driver: "postgres"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("List with an unknown driver: %v, want ErrInvalid", err)
	}
	if _, err := NewList(f.deps).Run(context.Background(), ListInput{Cursor: "nope"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("List with a bad cursor: %v, want ErrInvalid", err)
	}
}

// ---- update ----

func TestUpdateAppliesAPatch(t *testing.T) {
	f := newFixture()
	username := "reader"
	tls := "verify"

	updated, err := NewUpdate(f.deps).Run(context.Background(), UpdateInput{
		Ref: f.target.Ref, Patch: Patch{Username: &username, TLSMode: &tls}, Actor: "ops",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Username != "reader" || updated.TLSMode != "verify" {
		t.Fatalf("updated = %+v", updated)
	}

	// An empty patch is a mistake, not a no-op: it means the caller sent the wrong body.
	if _, err := NewUpdate(f.deps).Run(context.Background(), UpdateInput{
		Ref: f.target.Ref, Patch: Patch{}, Actor: "ops",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("empty patch: %v, want ErrInvalid", err)
	}

	// A patch that would make the target invalid is refused before it is written.
	bad := "not a name"
	if _, err := NewUpdate(f.deps).Run(context.Background(), UpdateInput{
		Ref: f.target.Ref, Patch: Patch{Name: &bad}, Actor: "ops",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid patch: %v, want ErrInvalid", err)
	}
}

// A rotated password is journaled as `rotated`, never as the value.
func TestUpdateRotatesThePassword(t *testing.T) {
	f := newFixture()
	password := "new-secret"

	updated, err := NewUpdate(f.deps).Run(context.Background(), UpdateInput{
		Ref: f.target.Ref, Patch: Patch{Password: &password}, Actor: "ops",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Secret != "sealed:new-secret" {
		t.Fatalf("secret = %q", updated.Secret)
	}
	detail := f.journal.entries[0].Detail
	if detail["password"] != "rotated" {
		t.Fatalf("journal detail = %v, want the rotation named and the value absent", detail)
	}
	if strings.Contains(strings.Join(values(detail), " "), "new-secret") {
		t.Fatal("the journal recorded the password")
	}
}

func values(detail map[string]string) []string {
	found := make([]string, 0, len(detail))
	for _, value := range detail {
		found = append(found, value)
	}
	return found
}

// ---- archive ----

func TestArchiveIsOnceOnly(t *testing.T) {
	f := newFixture()
	archived, err := NewArchive(f.deps).Run(context.Background(), ArchiveInput{Ref: f.target.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !archived.Archived() {
		t.Fatal("Archive returned a live target")
	}

	// The fixture's store now holds the archived row, so a second call sees it.
	if _, err := NewArchive(f.deps).Run(context.Background(), ArchiveInput{Ref: f.target.Ref}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second Archive: %v, want ErrConflict", err)
	}
}

// ---- probe ----

func TestProbeStoresWhatItMeasured(t *testing.T) {
	f := newFixture()
	f.driver.caps = domain.Capabilities{Version: "8.4.2"}

	out, err := NewProbe(f.deps).Run(context.Background(), ProbeInput{Ref: f.target.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if out.Target.ServerVersion != "8.4.2" || out.Target.LastProbeStatus != "ok" {
		t.Fatalf("probe = %+v", out.Target)
	}
	if out.Target.LastProbeAt == nil {
		t.Fatal("the probe did not record when it happened")
	}
}

// A refused connection is stored and returned: the caller learns that the target is unreachable,
// and the state file remembers the attempt.
func TestProbeStoresAFailure(t *testing.T) {
	f := newFixture()
	f.driver.probeErr = errors.New("dial tcp: connection refused")

	out, err := NewProbe(f.deps).Run(context.Background(), ProbeInput{Ref: f.target.Ref, Actor: "ops"})
	if err == nil {
		t.Fatal("Probe swallowed the failure")
	}
	if out.Target.LastProbeStatus != "failed" {
		t.Fatalf("status = %q, want failed", out.Target.LastProbeStatus)
	}
	if len(f.journal.entries) != 1 || f.journal.entries[0].Kind != "probed" {
		t.Fatalf("journal = %+v, want one `probed` line", f.journal.entries)
	}
}

// A row sealed with the previous key is re-sealed the moment the plaintext is in hand.
func TestProbeResealsAStalePassword(t *testing.T) {
	f := newFixture()
	f.sealer.stale = true

	out, err := NewProbe(f.deps).Run(context.Background(), ProbeInput{Ref: f.target.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if out.Target.Secret != "sealed:pw" {
		t.Fatalf("secret = %q, want it re-sealed with the current key", out.Target.Secret)
	}
}

func TestProbeRefusesAnArchivedTarget(t *testing.T) {
	f := newFixture()
	when := time.Now().UTC()
	f.targets.target.ArchivedAt = &when

	if _, err := NewProbe(f.deps).Run(context.Background(), ProbeInput{Ref: f.target.Ref}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Probe on an archived target: %v, want ErrConflict", err)
	}
}
