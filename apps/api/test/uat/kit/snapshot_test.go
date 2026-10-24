package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScrubMasksVolatileFields(t *testing.T) {
	body := []byte(`{"id":"abc","changeId":"def","createdAt":"2026-01-01T00:00:00Z","estDurationMs":42,"estAffected":3,"scratchSchema":"noxis_verify_deadbeef","status":"validated","checks":[{"target":"noxis_verify_cafe","ok":true}]}`)
	got, err := scrubJSON(body)
	if err != nil {
		t.Fatalf("scrubJSON: %v", err)
	}
	s := string(got)
	for _, want := range []string{`"<id>"`, `"<ts>"`, `"estDurationMs": 0`, `"<scratch>"`, `"status": "validated"`, `"estAffected": 3`} {
		if !strings.Contains(s, want) {
			t.Errorf("scrubbed body missing %q:\n%s", want, s)
		}
	}
	for _, leak := range []string{"abc", "deadbeef", "cafe"} {
		if strings.Contains(s, leak) {
			t.Errorf("scrubbed body still carries %q:\n%s", leak, s)
		}
	}
}

func TestScrubMasksSecrets(t *testing.T) {
	body := []byte(`{"token":"raw-token","password":"hunter2","tokenHash":"abc123","clientSecret":"s3cret","headers":{"Authorization":"Bearer x","Cookie":"a=b"},"username":"root"}`)
	got, err := scrubJSON(body)
	if err != nil {
		t.Fatalf("scrubJSON: %v", err)
	}
	s := string(got)
	for _, leak := range []string{"raw-token", "hunter2", "abc123", "s3cret", "Bearer x", "a=b"} {
		if strings.Contains(s, leak) {
			t.Errorf("scrubbed body still carries %q:\n%s", leak, s)
		}
	}
	if !strings.Contains(s, `"<secret>"`) || !strings.Contains(s, `"username": "root"`) {
		t.Errorf("scrubbed body = %s", s)
	}
}

func TestMismatchNamesTheFirstDifferingLine(t *testing.T) {
	dir := t.TempDir()
	if err := (Snapshots{Dir: dir, Update: true}).Match("x", []byte(`{"status":"validated","notes":"a"}`)); err != nil {
		t.Fatal(err)
	}
	err := (Snapshots{Dir: dir}).Match("x", []byte(`{"status":"applied","notes":"a"}`))
	if err == nil || !strings.Contains(err.Error(), `"status"`) {
		t.Fatalf("mismatch error = %v, want the differing line", err)
	}
}

func TestMatchWritesThenCompares(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"id":"one","status":"validated","estDurationMs":5}`)
	if err := (Snapshots{Dir: dir, Update: true}).Match("change", body); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "change.json")); err != nil {
		t.Fatalf("golden not written: %v", err)
	}
	// Same shape, different volatile values: still a match.
	same := []byte(`{"id":"two","status":"validated","estDurationMs":99}`)
	if err := (Snapshots{Dir: dir}).Match("change", same); err != nil {
		t.Errorf("expected a match: %v", err)
	}
	// A stable field changed: mismatch.
	changed := []byte(`{"id":"two","status":"applied","estDurationMs":99}`)
	if err := (Snapshots{Dir: dir}).Match("change", changed); err == nil {
		t.Error("expected a mismatch")
	}
	// A missing golden points at the fix.
	err := (Snapshots{Dir: dir}).Match("absent", body)
	if err == nil || !strings.Contains(err.Error(), "UPDATE_SNAPSHOTS=1") {
		t.Errorf("missing golden error = %v, want an UPDATE_SNAPSHOTS hint", err)
	}
}
