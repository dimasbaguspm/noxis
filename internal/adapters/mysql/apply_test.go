package mysql

import (
	"testing"

	"noxis/internal/common/sqltext"
	"noxis/internal/domain"
)

// The stage name is the only place the server says which algorithm it chose, so the mapping is
// worth pinning: guessing here would put the wrong word on a receipt.
func TestAlgorithmOf(t *testing.T) {
	for _, tc := range []struct {
		stage string
		want  domain.Algorithm
	}{
		{"stage/innodb/alter table (copy to tmp table)", domain.AlgorithmCopy},
		{"Copying to tmp table", domain.AlgorithmCopy},
		{"stage/innodb/alter table (instant)", domain.AlgorithmInstant},
		{"stage/innodb/alter table (in place)", domain.AlgorithmInPlace},
		{"stage/innodb/alter table (rebuild)", domain.AlgorithmInPlace},
		{"", ""},
		{"Sending data", ""},
	} {
		if got := algorithmOf(tc.stage); got != tc.want {
			t.Fatalf("algorithmOf(%q) = %q, want %q", tc.stage, got, tc.want)
		}
	}
}

// A guess is allowed before the statement runs, and it must never claim more than it knows:
// anything unrecognised is unknown, not INPLACE.
func TestExpectedAlgorithm(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want domain.Algorithm
	}{
		{"ALTER TABLE t ADD COLUMN c INT", domain.AlgorithmInstant},
		{"ALTER TABLE t DROP COLUMN c", domain.AlgorithmInstant},
		{"CREATE INDEX i ON t (c)", domain.AlgorithmInPlace},
		{"ALTER TABLE t ADD KEY i (c)", domain.AlgorithmInPlace},
		{"ALTER TABLE t ENGINE=InnoDB", ""},
	} {
		statement, err := sqltext.Classify(tc.raw)
		if err != nil {
			t.Fatalf("classify %q: %v", tc.raw, err)
		}
		if got := expectedAlgorithm(statement); got != tc.want {
			t.Fatalf("expectedAlgorithm(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
