package secret

import (
	"strings"
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	c, err := New(strings.Repeat("ab", 32)) // 64 hex chars
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	sealed, err := c.Encrypt("s3cret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !Encrypted(sealed) || strings.Contains(sealed, "s3cret") {
		t.Fatalf("sealed = %q", sealed)
	}
	plain, err := c.Decrypt(sealed)
	if err != nil || plain != "s3cret" {
		t.Fatalf("decrypt = %q, %v", plain, err)
	}
}

func TestCipherLegacyAndNoKey(t *testing.T) {
	// plaintext rows predate the key and stay readable
	if got, err := (*Cipher)(nil).Decrypt("plaintext"); err != nil || got != "plaintext" {
		t.Fatalf("legacy = %q, %v", got, err)
	}
	// encrypted data with no key must fail loudly, not return garbage
	if _, err := (*Cipher)(nil).Decrypt("enc:v1:abc"); err == nil {
		t.Fatal("encrypted value without a key must error")
	}
	// a different key cannot open the value
	a, _ := New(strings.Repeat("ab", 32))
	b, _ := New(strings.Repeat("cd", 32))
	sealed, _ := a.Encrypt("x")
	if _, err := b.Decrypt(sealed); err == nil {
		t.Fatal("wrong key must not decrypt")
	}
	if _, err := New("short"); err == nil {
		t.Fatal("bad key length must error")
	}
}
