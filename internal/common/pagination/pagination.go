// Package pagination is the pure paging logic for list endpoints. It takes
// primitive values (the raw query strings) so it stays free of net/http and
// reusable outside the transport.
package pagination

import (
	"strconv"

	"noxis/internal/constants"
)

// Params parses the list paging query values. page and perPage default to 1
// and defaultPerPage; perPage is capped so one request cannot pull the whole
// table.
func Params(pageValue, perPageValue string, defaultPerPage int) (page, perPage int) {
	page, _ = strconv.Atoi(pageValue)
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(perPageValue)
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > constants.MaxPageSize {
		perPage = constants.MaxPageSize
	}
	return page, perPage
}

// Slice is the page view of a full list; an out-of-range page yields an empty
// slice rather than an error.
func Slice[T any](all []T, page, perPage int) []T {
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
