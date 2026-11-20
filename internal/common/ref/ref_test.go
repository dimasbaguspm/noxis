package ref

import (
	"strings"
	"testing"
	"time"
)

func TestNewIsCanonical(t *testing.T) {
	got := New()
	if len(got) != Length {
		t.Fatalf("len = %d, want %d (%q)", len(got), Length, got)
	}
	if !Valid(got) {
		t.Fatalf("Valid(%q) = false", got)
	}
}

func TestNewAtSortsByTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	early := NewAt(base)
	late := NewAt(base.Add(time.Hour))
	if early >= late {
		t.Fatalf("refs do not sort by creation time: %q >= %q", early, late)
	}
}

func TestNewAtStampsMilliseconds(t *testing.T) {
	at := time.Date(2026, 5, 4, 3, 2, 1, 0, time.UTC)
	ms := uint64(at.UnixMilli())
	dashless := strings.ReplaceAll(NewAt(at), "-", "")
	if got := dashless[0:12]; got != hexOf(ms>>40, ms>>32, ms>>24, ms>>16, ms>>8, ms) {
		t.Fatalf("timestamp = %q", got)
	}
	if version := dashless[12]; version != '7' {
		t.Fatalf("version nibble = %q, want 7", version)
	}
}

func TestValid(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"uuid4", "3f6b1c0e-1a2b-4c3d-8e9f-0a1b2c3d4e5f", false},
		{"no dashes", "0197a1b2c3d47e8f9a0b1c2d3e4f5a6b", false},
		{"uppercase", "0197A1B2-C3D4-7E8F-9A0B-1C2D3E4F5A6B", false},
		{"bad variant", "0197a1b2-c3d4-7e8f-1a0b-1c2d3e4f5a6b", false},
		{"integer", "42", false},
		{"good", "0197a1b2-c3d4-7e8f-9a0b-1c2d3e4f5a6b", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.in); got != tc.want {
				t.Fatalf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// hexOf renders bytes as lowercase hex, for comparing against the textual ref.
func hexOf(bytes ...uint64) string {
	out := make([]byte, 0, len(bytes)*2)
	for _, b := range bytes {
		out = append(out, hexDigits[(b>>4)&0x0f], hexDigits[b&0x0f])
	}
	return string(out)
}
