package domain_test

import (
	"errors"
	"noxis/internal/domain"
	"strings"
	"testing"
)

// The mistake this error answers is copying the NAME a table prints, so it has to name the format and
// show one: "is not a reference" alone leaves the reader holding a value that looks perfectly valid.
func TestNotAReferenceSaysWhatAReferenceIs(t *testing.T) {
	err := domain.NotAReference("snaptojson-test")

	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("err = %v, want it to wrap ErrInvalid so the transport answers 400", err)
	}
	for _, want := range []string{"is not a reference", "full UUID", "01a1fd59-6d1e-7e0d-8784-de451d9156c7"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err, want)
		}
	}
}
