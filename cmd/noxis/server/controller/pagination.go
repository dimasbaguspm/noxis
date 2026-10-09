package controller

import (
	"net/http"
	"strconv"
)

// readPagination parses the list paging query params. page and perPage default
// to 1 and 20, perPage is capped so one request cannot pull the whole table.
func readPagination(r *http.Request, defaultPerPage int) (page, perPage int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(r.URL.Query().Get("perPage"))
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// pageSlice is the page view of a full list; an out-of-range page yields an
// empty slice rather than an error.
func pageSlice[T any](all []T, page, perPage int) []T {
	start := (page - 1) * perPage
	if start >= len(all) {
		return []T{}
	}
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}
	return all[start:end]
}