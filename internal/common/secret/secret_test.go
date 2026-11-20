package secret

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
)

func newKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return key
}

func TestSealOpenRoundTrip(t *testing.T) {
	k, err := New(newKey(t), nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sealed, err := k.Seal("ref-a", []byte("s3cret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed == "s3cret" {
		t.Fatal("Seal returned the plaintext")
	}
	plain, stale, err := k.Open("ref-a", sealed)
	if err != nil || stale || string(plain) != "s3cret" {
		t.Fatalf("Open = %q, %v, %v", plain, stale, err)
	}
	if _, _, err := k.Open("ref-b", sealed); !errors.Is(err, ErrOpen) {
		t.Fatalf("Open with a different aad = %v, want ErrOpen", err)
	}
}

func TestRotationReportsStale(t *testing.T) {
	old := newKey(t)
	before, err := mustNew(t, old, nil).Seal("ref", []byte("pw"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	fresh := newKey(t)
	ring := mustNew(t, fresh, old)
	plain, stale, err := ring.Open("ref", before)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if string(plain) != "pw" || !stale {
		t.Fatalf("Open = %q, stale %v; want %q, true", plain, stale, "pw")
	}
	// The current key writes, so a re-sealed value is no longer stale.
	sealed, err := ring.Seal("ref", plain)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, stale, err := ring.Open("ref", sealed); err != nil || stale {
		t.Fatalf("re-sealed Open = stale %v, %v; want false, nil", stale, err)
	}
}

func TestNoKey(t *testing.T) {
	k, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if k.Enabled() {
		t.Fatal("Enabled = true with no key")
	}
	if _, err := k.Seal("ref", []byte("x")); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Seal = %v, want ErrNoKey", err)
	}
}

func TestOpenRejectsGarbage(t *testing.T) {
	k := mustNew(t, newKey(t), nil)
	for _, sealed := range []string{"", "s3cret", "v2:AAAA", "v1:not-base64!!"} {
		if _, _, err := k.Open("ref", sealed); err == nil {
			t.Fatalf("Open(%q) accepted garbage", sealed)
		}
	}
}

func TestNewRejectsWrongLength(t *testing.T) {
	if _, err := New([]byte("short"), nil); err == nil {
		t.Fatal("New accepted a key that is not 32 bytes")
	}
}

func TestParseKey(t *testing.T) {
	key := newKey(t)
	for name, raw := range map[string]string{
		"hex":    hex.EncodeToString(key),
		"base64": base64.StdEncoding.EncodeToString(key),
		"raw":    base64.RawURLEncoding.EncodeToString(key),
	} {
		got, err := ParseKey(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(got) != string(key) {
			t.Fatalf("%s: key round trip differs", name)
		}
	}
	if _, err := ParseKey(""); !errors.Is(err, ErrNoKey) {
		t.Fatalf("ParseKey(\"\") = %v, want ErrNoKey", err)
	}
	if _, err := ParseKey("nope"); err == nil {
		t.Fatal("ParseKey accepted a short key")
	}
}

func mustNew(t *testing.T, current, previous []byte) *Keyring {
	t.Helper()
	k, err := New(current, previous)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return k
}
