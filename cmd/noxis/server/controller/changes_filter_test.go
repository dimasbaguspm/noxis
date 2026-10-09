package controller

import (
	"testing"

	"noxis/internal/entity"
)

// The queue filter is a case-insensitive substring on the connection name; an
// empty needle must keep every change.
func TestFilterByConnection(t *testing.T) {
	changes := []entity.Change{{Connection: "prod-db"}, {Connection: "cache"}, {Connection: "prod-cache"}}
	cases := []struct {
		needle string
		want   int
	}{
		{"", 3},
		{"PROD", 2},
		{"cache", 2},
		{"nope", 0},
	}
	for _, tc := range cases {
		if got := filterByConnection(changes, tc.needle); len(got) != tc.want {
			t.Fatalf("filter %q = %d, want %d", tc.needle, len(got), tc.want)
		}
	}
}
