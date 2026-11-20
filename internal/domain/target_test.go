package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"noxis/internal/domain"
)

func fullTarget() domain.Target {
	return domain.Target{
		Name: "prod", Driver: "mysql", Host: "db.internal", Port: 3306,
		Database: "app", Username: "noxis", TLSMode: "preferred", CreatedBy: "ops",
	}
}

// Every rule here refuses a target before it reaches the state file, so each one is a test.
func TestTargetValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Target)
		want   string
	}{
		{"ok", func(*domain.Target) {}, ""},
		{"short name", func(t *domain.Target) { t.Name = "a" }, "name"},
		{"uppercase name", func(t *domain.Target) { t.Name = "Prod" }, "name"},
		{"name with a space", func(t *domain.Target) { t.Name = "prod east" }, "name"},
		{"unknown driver", func(t *domain.Target) { t.Driver = "postgres" }, "driver"},
		{"no host", func(t *domain.Target) { t.Host = "  " }, "host"},
		{"port zero", func(t *domain.Target) { t.Port = 0 }, "1-65535"},
		{"port too high", func(t *domain.Target) { t.Port = 65536 }, "1-65535"},
		{"no database", func(t *domain.Target) { t.Database = "" }, "database"},
		{"no username", func(t *domain.Target) { t.Username = "" }, "username"},
		{"unknown tls mode", func(t *domain.Target) { t.TLSMode = "maybe" }, "tls mode"},
		{"no creator", func(t *domain.Target) { t.CreatedBy = "" }, "caller"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subject := fullTarget()
			tc.change(&subject)
			err := subject.Validate()
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("Validate: %v, want nil", err)
			case tc.want != "" && err == nil:
				t.Fatalf("Validate accepted %+v", subject)
			case tc.want != "" && !errors.Is(err, domain.ErrInvalid):
				t.Fatalf("Validate: %v, want ErrInvalid", err)
			case tc.want != "" && !strings.Contains(err.Error(), tc.want):
				t.Fatalf("Validate: %q does not mention %q", err, tc.want)
			}
		})
	}
}

// An archived target is put away, never removed: the journal still points at it, so the predicate
// that says "gone" has to be the timestamp and not a missing row.
func TestTargetArchived(t *testing.T) {
	target := fullTarget()
	if target.Archived() {
		t.Fatal("a fresh target is reported as archived")
	}
	when := time.Now().UTC()
	target.ArchivedAt = &when
	if !target.Archived() {
		t.Fatal("a target with an archived_at is not reported as archived")
	}
}
