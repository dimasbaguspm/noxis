package entity

import "testing"

func TestLevelAllows(t *testing.T) {
	cases := []struct {
		have, need string
		want       bool
	}{
		{LevelRead, LevelRead, true},
		{LevelRead, LevelWrite, false},
		{LevelWrite, LevelRead, true},
		{LevelWrite, LevelAdmin, false},
		{LevelAdmin, LevelWrite, true},
		{LevelAdmin, LevelAdmin, true},
		{"", LevelRead, false},
		{LevelRead, "bogus", false},
	}
	for _, tc := range cases {
		if got := LevelAllows(tc.have, tc.need); got != tc.want {
			t.Errorf("LevelAllows(%q, %q)=%v want %v", tc.have, tc.need, got, tc.want)
		}
	}
}
