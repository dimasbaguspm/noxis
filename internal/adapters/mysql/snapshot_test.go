package mysql

import (
	"strings"
	"testing"
)

// A snapshot compares the shape of a table, not its history: the AUTO_INCREMENT counter moves on its
// own every time a row is inserted, so a rehearsal that kept it would report a schema change nobody
// made. The counter is removed wherever it appears in a line, and the trailing space it leaves behind
// goes with it.
func TestNormaliseDDLRemovesTheCounter(t *testing.T) {
	for _, tc := range []struct {
		name, ddl, want string
	}{
		{
			name: "the counter at the end of a line",
			ddl:  "CREATE TABLE `t` (\n  `id` bigint NOT NULL AUTO_INCREMENT\n) ENGINE=InnoDB AUTO_INCREMENT=42 DEFAULT CHARSET=utf8mb4",
			want: "CREATE TABLE `t` (\n  `id` bigint NOT NULL AUTO_INCREMENT\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		},
		{
			name: "a counter with nothing after it",
			ddl:  "CREATE TABLE `t` (...) ENGINE=InnoDB AUTO_INCREMENT=7",
			want: "CREATE TABLE `t` (...) ENGINE=InnoDB",
		},
		{
			name: "trailing spaces go, they are not a difference either",
			ddl:  "CREATE TABLE `t` (\n  `id` int\n)   ",
			want: "CREATE TABLE `t` (\n  `id` int\n)",
		},
		{
			name: "a DDL without a counter is untouched",
			ddl:  "CREATE TABLE `t` (\n  `id` int\n) ENGINE=InnoDB",
			want: "CREATE TABLE `t` (\n  `id` int\n) ENGINE=InnoDB",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normaliseDDL(tc.ddl); got != tc.want {
				t.Fatalf("normaliseDDL:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

// sameSchema is the verdict a rehearsal stands on: if the rollback put the schema back exactly, it
// returns true; a table added, removed or changed is false. An empty schema compared to an empty one
// is a match, and a snapshot that lost a table is never a match with one that had it.
func TestSameSchema(t *testing.T) {
	for _, tc := range []struct {
		name          string
		before, after map[string]string
		want          bool
	}{
		{"both empty", map[string]string{}, map[string]string{}, true},
		{"identical", map[string]string{"users": "CREATE TABLE users"}, map[string]string{"users": "CREATE TABLE users"}, true},
		{"a table the rollback did not put back", map[string]string{"users": "CREATE TABLE users"}, map[string]string{}, false},
		{
			name:   "a table of the same name, different shape",
			before: map[string]string{"users": "CREATE TABLE users"},
			after:  map[string]string{"users": "CREATE TABLE users_v2"},
			want:   false,
		},
		{
			name:   "a table the rollback left behind",
			before: map[string]string{},
			after:  map[string]string{"users": "CREATE TABLE users"},
			want:   false,
		},
		{
			name:   "a different name in its place",
			before: map[string]string{"users": "CREATE TABLE users"},
			after:  map[string]string{"users_v2": "CREATE TABLE users"},
			want:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameSchema(tc.before, tc.after); got != tc.want {
				t.Fatalf("sameSchema = %v, want %v", got, tc.want)
			}
		})
	}
}

// The diff is what the reviewer reads instead of the table list, so every kind of change has to be
// named in it: a removed table with its old DDL, an added table with its new one, and a changed table
// line by line. Two identical snapshots have to say so in words rather than print nothing, and a line
// that survived must not be printed as a change.
func TestDiffSchemas(t *testing.T) {
	for _, tc := range []struct {
		name          string
		before, after map[string]string
		want          []string
		absent        []string
	}{
		{
			name:   "no change is said in words",
			before: map[string]string{"users": "CREATE TABLE `users` (\n  `id` int\n)"},
			after:  map[string]string{"users": "CREATE TABLE `users` (\n  `id` int\n)"},
			want:   []string{"no schema change"},
			absent: []string{"changed table", "added table", "removed table"},
		},
		{
			name:   "a removed table carries its old DDL",
			before: map[string]string{"old": "CREATE TABLE `old` (\n  `id` int\n)"},
			after:  map[string]string{},
			want:   []string{"removed table old", "- CREATE TABLE `old` (", "-   `id` int"},
		},
		{
			name:   "an added table carries its new DDL",
			before: map[string]string{},
			after:  map[string]string{"fresh": "CREATE TABLE `fresh` (\n  `id` int\n)"},
			want:   []string{"added table fresh", "+ CREATE TABLE `fresh` (", "+   `id` int"},
		},
		{
			name:   "a changed table is diffed line by line",
			before: map[string]string{"users": "CREATE TABLE `users` (\n  `id` int\n)"},
			after:  map[string]string{"users": "CREATE TABLE `users` (\n  `id` int\n  `note` text\n)"},
			want:   []string{"changed table users", "+   `note` text"},
			absent: []string{"- ", "added table", "removed table"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := diffSchemas(tc.before, tc.after)
			for _, phrase := range tc.want {
				if !strings.Contains(got, phrase) {
					t.Fatalf("diff does not contain %q:\n%s", phrase, got)
				}
			}
			for _, phrase := range tc.absent {
				if strings.Contains(got, phrase) {
					t.Fatalf("diff contains %q, which is not a difference:\n%s", phrase, got)
				}
			}
		})
	}
}

// Tables are diffed in the order a reader can follow, whatever order the map handed them over: the
// union of both snapshots is sorted, and a name in both appears once.
func TestUnionKeysIsSortedAndDeduplicated(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b map[string]string
		want []string
	}{
		{"both empty", nil, nil, nil},
		{"shared names appear once", map[string]string{"b": ""}, map[string]string{"b": "", "a": ""}, []string{"a", "b"}},
		{"only the first snapshot", map[string]string{"z": "", "m": ""}, map[string]string{}, []string{"m", "z"}},
		{"only the second snapshot", map[string]string{}, map[string]string{"z": "", "m": ""}, []string{"m", "z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := unionKeys(tc.a, tc.b)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("unionKeys = %v, want %v", got, tc.want)
			}
		})
	}
}

// The line diff is what makes a changed table readable, so a line that survived is not printed and a
// line that arrived is printed with a plus. Old lines come first, then new ones.
func TestLineDiff(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		want                []string
	}{
		{"nothing changed", "a\nb", "a\nb", nil},
		{"a line removed", "a\nb", "a", []string{"- b"}},
		{"a line added", "a", "a\nb", []string{"+ b"}},
		{"a line replaced", "a", "b", []string{"- a", "+ b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := lineDiff(tc.before, tc.after)
			if len(tc.want) == 0 {
				if got != "" {
					t.Fatalf("lineDiff = %q, want nothing", got)
				}
				return
			}
			if want := strings.Join(tc.want, "\n") + "\n"; got != want {
				t.Fatalf("lineDiff = %q, want %q", got, want)
			}
		})
	}
}
