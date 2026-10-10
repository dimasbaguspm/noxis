package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageSlice(t *testing.T) {
	all := []int{1, 2, 3, 4, 5}
	if got := pageSlice(all, 1, 2); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("page 1 = %v", got)
	}
	if got := pageSlice(all, 3, 2); len(got) != 1 || got[0] != 5 {
		t.Fatalf("page 3 = %v", got)
	}
	if got := pageSlice(all, 9, 2); len(got) != 0 {
		t.Fatalf("out of range = %v", got)
	}
}

func TestReadPagination(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/x?page=2&perPage=7", http.NoBody)
	if page, per := readPagination(r, 20); page != 2 || per != 7 {
		t.Fatalf("got %d/%d", page, per)
	}
	// defaults and the cap
	r = httptest.NewRequest("GET", "/v1/x", http.NoBody)
	if page, per := readPagination(r, 20); page != 1 || per != 20 {
		t.Fatalf("defaults %d/%d", page, per)
	}
	r = httptest.NewRequest("GET", "/v1/x?perPage=999", http.NoBody)
	if _, per := readPagination(r, 20); per != 100 {
		t.Fatalf("cap = %d", per)
	}
}
