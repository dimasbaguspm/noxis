package pagination

import "testing"

func TestSlice(t *testing.T) {
	all := []int{1, 2, 3, 4, 5}
	if got := Slice(all, 1, 2); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("page 1 = %v", got)
	}
	if got := Slice(all, 3, 2); len(got) != 1 || got[0] != 5 {
		t.Fatalf("page 3 = %v", got)
	}
	if got := Slice(all, 9, 2); len(got) != 0 {
		t.Fatalf("out of range = %v", got)
	}
}

func TestParams(t *testing.T) {
	if page, per := Params("2", "7", 20); page != 2 || per != 7 {
		t.Fatalf("got %d/%d", page, per)
	}
	if page, per := Params("", "", 20); page != 1 || per != 20 {
		t.Fatalf("defaults %d/%d", page, per)
	}
	if _, per := Params("", "999", 20); per != 100 {
		t.Fatalf("cap = %d", per)
	}
}