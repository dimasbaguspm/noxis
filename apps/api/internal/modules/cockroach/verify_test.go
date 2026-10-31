package cockroach

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreatedTables(t *testing.T) {
	cases := []struct {
		sql  string
		want []string
	}{
		{"CREATE TABLE notes (id INT)", []string{"notes"}},
		{"CREATE TABLE IF NOT EXISTS public.notes (id INT)", []string{"public.notes"}},
		{"CREATE OR REPLACE TABLE t (id INT)", []string{"t"}},
		{"CREATE TEMPORARY TABLE tmp (id INT)", []string{"tmp"}},
		{"CREATE INDEX idx ON notes (id)", nil},
		{"ALTER TABLE notes ADD COLUMN tag TEXT", nil},
		{"CREATE TABLE a (id INT); CREATE TABLE b (id INT)", []string{"a", "b"}},
	}
	for _, tc := range cases {
		got := mapKeys(createdTables(tc.sql))
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: createdTables = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

func TestQualifiedIdent(t *testing.T) {
	for _, sql := range []string{
		"DROP TABLE public.victim",
		"UPDATE t SET t.name = 'x'",
		"ALTER TABLE `demo`.`items` ADD COLUMN x INT",
	} {
		if !hasQualifiedIdent(sql) {
			t.Errorf("hasQualifiedIdent(%q) = false", sql)
		}
	}
	for _, sql := range []string{
		"DROP TABLE victim",
		"UPDATE items SET name = 'x'",
		"SELECT 1",
	} {
		if hasQualifiedIdent(sql) {
			t.Errorf("hasQualifiedIdent(%q) = true", sql)
		}
	}
}

func mapKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestReplayRetriesALostRace proves a contended dry run is replayed instead of recorded as a failed change.
func TestReplayRetriesALostRace(t *testing.T) {
	calls := 0
	res, err := replay(func() (Result, error) {
		calls++
		if calls == 1 {
			return Result{}, fmt.Errorf("query: %w", &pgconn.PgError{Code: retryableCode})
		}
		return Result{Message: "proved"}, nil
	})
	if err != nil || res.Message != "proved" || calls != 2 {
		t.Fatalf("replay = %+v, %v after %d calls", res, err, calls)
	}
}

// TestReplayGivesUpWhenTheRaceNeverClears proves the budget is bounded and the engine's own reason survives.
func TestReplayGivesUpWhenTheRaceNeverClears(t *testing.T) {
	calls := 0
	_, err := replay(func() (Result, error) {
		calls++
		return Result{}, &pgconn.PgError{Code: retryableCode}
	})
	if err == nil || calls != retryBudget {
		t.Fatalf("replay ran %d times, err = %v", calls, err)
	}
}

// TestReplayDoesNotRetryAnotherError proves a real failure is reported on the first attempt.
func TestReplayDoesNotRetryAnotherError(t *testing.T) {
	calls := 0
	_, err := replay(func() (Result, error) {
		calls++
		return Result{}, errors.New("rollback did not restore public.t")
	})
	if err == nil || calls != 1 {
		t.Fatalf("replay ran %d times, err = %v", calls, err)
	}
}
