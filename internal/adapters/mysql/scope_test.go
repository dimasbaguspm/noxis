package mysql

import (
	"errors"
	"noxis/internal/domain"
	"strings"
	"testing"
)

// The scope is the list of tables a rehearsal rebuilds, so it decides what evidence a reviewer sees.
// A change is rehearsed against the schema the target governs and nothing else: a statement naming
// another schema is refused by name, because rewriting it would prove something about a database
// nobody approved touching.
func TestRehearsalScope(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   domain.Change
		database string
		want     []string
		wantErr  error
		says     []string
	}{
		{
			name:     "a bare table is taken as it is",
			change:   domain.Change{Query: "ALTER TABLE users ADD COLUMN note TEXT"},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "a table qualified with the governed schema is the same table",
			change:   domain.Change{Query: "ALTER TABLE app.users ADD COLUMN note TEXT"},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "the schema is compared without case",
			change:   domain.Change{Query: "ALTER TABLE APP.users ADD COLUMN note TEXT"},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "backticks are stripped, not carried into the scope",
			change:   domain.Change{Query: "ALTER TABLE `app`.`users` ADD COLUMN note TEXT"},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "a table in another schema is refused, naming both schemas",
			change:   domain.Change{Query: "ALTER TABLE billing.invoices ADD COLUMN note TEXT"},
			database: "app",
			wantErr:  domain.ErrUnsupported,
			says:     []string{"billing", "app"},
		},
		{
			name: "the rollback's tables are in scope too",
			change: domain.Change{
				Query:    "ALTER TABLE users ADD COLUMN note TEXT",
				Rollback: "ALTER TABLE audit_log DROP COLUMN note",
			},
			database: "app",
			want:     []string{"audit_log", "users"},
		},
		{
			name: "a table named twice appears once",
			change: domain.Change{
				Query:    "ALTER TABLE users ADD COLUMN note TEXT",
				Rollback: "ALTER TABLE users DROP COLUMN note",
			},
			database: "app",
			want:     []string{"users"},
		},
		{
			name: "a rollback in another schema is refused as well",
			change: domain.Change{
				Query:    "ALTER TABLE users ADD COLUMN note TEXT",
				Rollback: "ALTER TABLE archive.users DROP COLUMN note",
			},
			database: "app",
			wantErr:  domain.ErrUnsupported,
			says:     []string{"archive", "app"},
		},
		{
			name: "a rollback the parser cannot read adds nothing to the scope",
			change: domain.Change{
				Query:    "ALTER TABLE users ADD COLUMN note TEXT",
				Rollback: "ALTER TABLE users DROP COLUMN note; ALTER TABLE other DROP COLUMN note",
			},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "an empty rollback is not a table source",
			change:   domain.Change{Query: "ALTER TABLE users ADD COLUMN note TEXT", Rollback: "   "},
			database: "app",
			want:     []string{"users"},
		},
		{
			name:     "a statement that cannot be read is invalid, not unscoped",
			change:   domain.Change{Query: "   "},
			database: "app",
			wantErr:  domain.ErrInvalid,
		},
		{
			name:     "a statement naming no table has an empty scope",
			change:   domain.Change{Query: "SELECT 1"},
			database: "app",
			want:     nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rehearsalScope(tc.change, tc.database)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("rehearsalScope(%q) error = %v, want %v", tc.change.Query, err, tc.wantErr)
			}
			for _, phrase := range tc.says {
				if !strings.Contains(err.Error(), phrase) {
					t.Fatalf("error %q does not name %q", err, phrase)
				}
			}
			if tc.wantErr != nil {
				return
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("scope = %v, want %v", got, tc.want)
			}
		})
	}
}

// A change with no rollback is irreversible, and the verdict must follow the kind of statement: a
// SELECT with no rollback undoes nothing, so it is not irreversible; a DDL, a DML or a statement
// nobody can read is.
func TestIrreversible(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change domain.Change
		want   bool
	}{
		{"a DDL with no rollback", domain.Change{Query: "ALTER TABLE users ADD COLUMN note TEXT"}, true},
		{"an insert with no rollback", domain.Change{Query: "INSERT INTO users (id) VALUES (1)"}, true},
		{"an update with no rollback", domain.Change{Query: "UPDATE users SET note = ''"}, true},
		{"a delete with no rollback", domain.Change{Query: "DELETE FROM users WHERE id = 1"}, true},
		{"a select with no rollback", domain.Change{Query: "SELECT * FROM users"}, false},
		{
			name:   "a DDL with a rollback",
			change: domain.Change{Query: "ALTER TABLE users ADD COLUMN note TEXT", Rollback: "ALTER TABLE users DROP COLUMN note"},
			want:   false,
		},
		{
			name:   "a rollback of nothing but spaces is no rollback",
			change: domain.Change{Query: "ALTER TABLE users ADD COLUMN note TEXT", Rollback: "   "},
			want:   true,
		},
		{"a statement nobody can read", domain.Change{Query: ""}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := irreversible(tc.change); got != tc.want {
				t.Fatalf("irreversible(%q, rollback %q) = %v, want %v",
					tc.change.Query, tc.change.Rollback, got, tc.want)
			}
		})
	}
}
