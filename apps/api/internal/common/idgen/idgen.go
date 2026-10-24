// Package idgen generates correlation and identity ids (crypto/rand, no external uuid dependency): dashless uuids and fixed-length hex ids.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

const uuidLen = 16

// UUID returns a 32-char lowercase hex uuid without dashes.
func UUID() string { return Hex(uuidLen) }

// Hex returns n random bytes as lowercase hex (2n chars). crypto/rand does not fail in practice; a failure is unrecoverable, so it panics.
func Hex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("idgen: crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}
