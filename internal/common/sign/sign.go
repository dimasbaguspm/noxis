// Package sign provides HMAC-SHA256 sign/verify helpers (hex-encoded) for
// authenticating requests and payloads with a shared secret.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign returns hex hmac-sha256(secret, data).
func Sign(secret, data []byte) string {
	m := hmac.New(sha256.New, secret)
	_, _ = m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

// Verify reports whether sig (hex) was produced by Sign(secret, data).
func Verify(secret, data []byte, sig string) bool {
	return hmac.Equal([]byte(Sign(secret, data)), []byte(sig))
}
