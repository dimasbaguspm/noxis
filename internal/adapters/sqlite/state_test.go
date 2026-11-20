package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"noxis/internal/common/ref"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

func open(t *testing.T) *DB {
	t.Helper()
	state, err := Open(context.Background(), filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = state.Close() })
	return state
}

func newTarget(t *testing.T, name string) domain.Target {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	return domain.Target{
		Ref:       ref.New(),
		Name:      name,
		Driver:    "mysql",
		Host:      "127.0.0.1",
		Port:      3306,
		Database:  "app",
		Username:  "noxis",
		Secret:    "v1:sealed",
		TLSMode:   "preferred",
		Options:   map[string]string{"timeout": "5s"},
		CreatedBy: "ops",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestOpenSettlesSchemaOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	ctx := context.Background()
	first, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	version, err := first.userVersion(ctx)
	if err != nil {
		t.Fatalf("userVersion: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("user_version = %d, want %d", version, SchemaVersion)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	second, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	_ = second.Close()
}

func TestOpenRefusesAnotherSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if _, err := raw.ExecContext(context.Background(), "PRAGMA user_version = 9"); err != nil {
		t.Fatalf("stamp version: %v", err)
	}
	_ = raw.Close()

	if _, err := Open(context.Background(), path); err == nil {
		t.Fatal("Open accepted a state file with an unknown schema version")
	}
}

func TestOpenRefusesAForeignDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "other.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if _, err := raw.ExecContext(context.Background(), "CREATE TABLE someone_else (x TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	_ = raw.Close()

	if _, err := Open(context.Background(), path); err == nil {
		t.Fatal("Open adopted a database it did not write")
	}
}

func TestCreateAndByRef(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	target := newTarget(t, "primary")

	created, err := state.Create(ctx, target)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Ref != target.Ref {
		t.Fatalf("ref = %q, want %q", created.Ref, target.Ref)
	}

	found, err := state.ByRef(ctx, target.Ref)
	if err != nil {
		t.Fatalf("ByRef: %v", err)
	}
	if found.Name != target.Name || found.Secret != target.Secret || found.Options["timeout"] != "5s" {
		t.Fatalf("round trip lost a field: %+v", found)
	}
	if found.Archived() {
		t.Fatal("a fresh target reports itself archived")
	}
	if !found.CreatedAt.Equal(target.CreatedAt) {
		t.Fatalf("created_at = %s, want %s", found.CreatedAt, target.CreatedAt)
	}
}

func TestCreateRefusesADuplicateName(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	first := newTarget(t, "primary")
	if _, err := state.Create(ctx, first); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := state.Create(ctx, newTarget(t, "primary")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Create duplicate = %v, want ErrConflict", err)
	}
}

func TestByRefUnknownIsNotFound(t *testing.T) {
	state := open(t)
	if _, err := state.ByRef(context.Background(), ref.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ByRef = %v, want ErrNotFound", err)
	}
}

func TestListPagesByCursor(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	var refs []string
	for _, name := range []string{"one", "two", "three"} {
		created, err := state.Create(ctx, newTarget(t, name))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		refs = append(refs, created.Ref)
	}

	first, err := state.List(ctx, ports.TargetFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("first page has %d rows, want 2", len(first))
	}
	// Refs sort descending by construction, which is newest first to the millisecond; two
	// refs minted inside the same millisecond tie on time, so the expected order is the
	// sorted set rather than insertion order.
	want := append([]string(nil), refs...)
	sort.Sort(sort.Reverse(sort.StringSlice(want)))
	if first[0].Ref != want[0] || first[1].Ref != want[1] {
		t.Fatalf("page order = %q, %q; want %q, %q", first[0].Ref, first[1].Ref, want[0], want[1])
	}

	second, err := state.List(ctx, ports.TargetFilter{Limit: 2, Cursor: first[1].Ref})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(second) != 1 || second[0].Ref != want[2] {
		t.Fatalf("second page = %+v", second)
	}
}

func TestListFiltersAndHidesArchived(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	archived, err := state.Create(ctx, newTarget(t, "gone"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := state.Create(ctx, newTarget(t, "kept")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	now := time.Now().UTC()
	archived.ArchivedAt = &now
	archived.UpdatedAt = now
	if _, err := state.Update(ctx, archived); err != nil {
		t.Fatalf("Update: %v", err)
	}

	rows, err := state.List(ctx, ports.TargetFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "kept" {
		t.Fatalf("listed %+v, want only the live target", rows)
	}

	all, err := state.List(ctx, ports.TargetFilter{Limit: 10, IncludeArchived: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("include_archived listed %d rows, want 2", len(all))
	}

	// A typed % is a literal, not a wildcard.
	if matches, err := state.List(ctx, ports.TargetFilter{Limit: 10, Query: "%"}); err != nil || len(matches) != 0 {
		t.Fatalf("LIKE escaping: %d matches, %v", len(matches), err)
	}
}

func TestUpdateMissingIsNotFound(t *testing.T) {
	state := open(t)
	target := newTarget(t, "ghost")
	if _, err := state.Update(context.Background(), target); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update = %v, want ErrNotFound", err)
	}
}

func TestJournalIsAppendOnly(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	target := newTarget(t, "primary")
	if _, err := state.Create(ctx, target); err != nil {
		t.Fatalf("Create: %v", err)
	}
	entry := domain.JournalEntry{
		Ref:         ref.New(),
		SubjectRef:  target.Ref,
		SubjectKind: "target",
		Kind:        "created",
		Actor:       "ops",
		Detail:      map[string]string{"name": target.Name},
		CreatedAt:   time.Now().UTC(),
	}
	if err := state.Append(ctx, entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := state.Entries(ctx, target.Ref, 10)
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Actor != "ops" || entries[0].Detail["name"] != target.Name {
		t.Fatalf("entries = %+v", entries)
	}

	// Updating the subject must not touch the journal: only an append can add a line.
	target.Username = "rotated"
	if _, err := state.Update(ctx, target); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if after, _ := state.Entries(ctx, target.Ref, 10); len(after) != 1 {
		t.Fatalf("journal grew to %d lines without an append", len(after))
	}
}

func TestProbeFieldsRoundTrip(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	target := newTarget(t, "probed")
	created, err := state.Create(ctx, target)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.LastProbeStatus != "" || created.LastProbeAt != nil {
		t.Fatalf("a fresh target reports a probe: %+v", created)
	}

	at := time.Now().UTC().Truncate(time.Millisecond)
	created.LastProbeAt = &at
	created.LastProbeStatus = "ok"
	created.ServerVersion = "11.4.2-MariaDB"
	if _, err := state.Update(ctx, created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := state.ByRef(ctx, created.Ref)
	if err != nil {
		t.Fatalf("ByRef: %v", err)
	}
	if found.ServerVersion != "11.4.2-MariaDB" || found.LastProbeStatus != "ok" {
		t.Fatalf("probe fields = %+v", found)
	}
	if found.LastProbeAt == nil || !found.LastProbeAt.Equal(at) {
		t.Fatalf("last_probe_at = %v, want %v", found.LastProbeAt, at)
	}
}

func newProposal(t *testing.T, targetRef, title string) domain.Proposal {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	return domain.Proposal{
		Ref: ref.New(), TargetRef: targetRef, Title: title, Kind: domain.KindDDL,
		Query: "ALTER TABLE users ADD COLUMN x INT", Rollback: "ALTER TABLE users DROP COLUMN x",
		Status: domain.StatusDraft, Tier: domain.TierShadow, ProposedBy: "ops",
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestProposalAndRehearsalRoundTrip(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	target, err := state.Create(ctx, newTarget(t, "primary"))
	if err != nil {
		t.Fatalf("Create target: %v", err)
	}
	proposal := newProposal(t, target.Ref, "add x")
	if _, err := state.CreateProposal(ctx, proposal); err != nil {
		t.Fatalf("CreateProposal: %v", err)
	}

	rehearsal := domain.Rehearsal{
		Ref: ref.New(), ProposalRef: proposal.Ref, Tier: domain.TierShadow,
		Status: domain.RehearsalFailed, SchemaDiff: "- a\n+ b", RowDiff: "users: 0 -> 0",
		Irreversible: true, Note: "the rollback left the schema different",
		DurationMS: 12, CreatedAt: time.Now().UTC(),
	}
	if _, err := state.CreateRehearsal(ctx, rehearsal); err != nil {
		t.Fatalf("CreateRehearsal: %v", err)
	}

	proposal.Status = domain.StatusFailed
	proposal.RehearsalRef = rehearsal.Ref
	proposal.Irreversible = true
	proposal.Failure = "the rollback left the schema different"
	proposal.UpdatedAt = time.Now().UTC()
	if _, err := state.UpdateProposal(ctx, proposal); err != nil {
		t.Fatalf("UpdateProposal: %v", err)
	}

	found, err := state.ProposalByRef(ctx, proposal.Ref)
	if err != nil {
		t.Fatalf("ProposalByRef: %v", err)
	}
	if found.Status != domain.StatusFailed || found.Kind != domain.KindDDL || !found.Irreversible {
		t.Fatalf("proposal round trip lost a field: %+v", found)
	}
	if found.RehearsalRef != rehearsal.Ref {
		t.Fatalf("rehearsal_ref = %q", found.RehearsalRef)
	}

	latest, err := state.LatestRehearsal(ctx, proposal.Ref)
	if err != nil {
		t.Fatalf("LatestRehearsal: %v", err)
	}
	if latest.SchemaDiff != "- a\n+ b" || !latest.Irreversible || latest.DurationMS != 12 {
		t.Fatalf("rehearsal round trip lost a field: %+v", latest)
	}

	page, err := state.ListProposals(ctx, ports.ProposalFilter{Limit: 10, Status: "failed", Mine: "ops"})
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(page) != 1 || page[0].Ref != proposal.Ref {
		t.Fatalf("list = %+v", page)
	}
	if empty, _ := state.ListProposals(ctx, ports.ProposalFilter{Limit: 10, Status: "verified"}); len(empty) != 0 {
		t.Fatalf("status filter let through %d rows", len(empty))
	}
}

func TestLatestRehearsalWithoutOne(t *testing.T) {
	state := open(t)
	if _, err := state.LatestRehearsal(context.Background(), ref.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("LatestRehearsal = %v, want ErrNotFound", err)
	}
}
