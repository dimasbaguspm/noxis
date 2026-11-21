// Package ref mints and validates the opaque reference every row is keyed by: a
// canonical UUIDv7. Two v7 refs sort by the moment they were created, which is what
// makes `ORDER BY ref DESC` both newest-first and a stable keyset cursor.
package ref

import (
	"crypto/rand"
	"strings"
	"time"
)

// Length is the canonical textual form, 8-4-4-4-12.
const Length = 36

const hexDigits = "0123456789abcdef"

// New mints a ref stamped with the current time.
func New() string { return NewAt(time.Now()) }

// NewAt mints a ref stamped with t. Tests that need a known order use it.
func NewAt(t time.Time) string {
	var b [16]byte
	ms := uint64(t.UnixMilli())
	b[0] = byte((ms >> 40) & 0xff)
	b[1] = byte((ms >> 32) & 0xff)
	b[2] = byte((ms >> 24) & 0xff)
	b[3] = byte((ms >> 16) & 0xff)
	b[4] = byte((ms >> 8) & 0xff)
	b[5] = byte(ms & 0xff)
	// crypto/rand does not fail in practice; a failure here is unrecoverable, so the
	// process stops instead of minting a duplicate reference.
	if _, err := rand.Read(b[6:]); err != nil {
		panic("ref: crypto/rand: " + err.Error())
	}
	b[6] = b[6]&0x0f | 0x70 // version 7
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return format(b)
}

// Valid reports whether s is a canonical UUIDv7: the only shape a path param or a
// cursor may take.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := range Length {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	if s[14] != '7' {
		return false
	}
	switch s[19] {
	case '8', '9', 'a', 'b':
		return true
	}
	return false
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f'
}

func format(b [16]byte) string {
	var sb strings.Builder
	sb.Grow(Length)
	for i, v := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			sb.WriteByte('-')
		}
		sb.WriteByte(hexDigits[v>>4])
		sb.WriteByte(hexDigits[v&0x0f])
	}
	return sb.String()
}
