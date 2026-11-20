// Package cursor keeps every list on a keyset: the ref of the last row on the page,
// never an offset. An offset answers "what is at position N now", which is a different
// question from "what comes after this row", and drifts as soon as anything is written.
package cursor

import (
	"fmt"
	"noxis/internal/common/ref"
)

// Limit clamps a requested page size, treating a non-positive request as "not given".
func Limit(n, fallback, ceiling int) int {
	switch {
	case n <= 0:
		return fallback
	case n > ceiling:
		return ceiling
	default:
		return n
	}
}

// Ref validates an incoming cursor. Empty means the first page.
func Ref(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if !ref.Valid(s) {
		return "", fmt.Errorf("cursor: %q is not a reference", s)
	}
	return s, nil
}

// Next names the following page from the last row that was read. It is empty when the
// page was not full: a full page is the only evidence that more rows exist.
func Next(lastRef string, full bool) string {
	if !full || lastRef == "" {
		return ""
	}
	return lastRef
}
