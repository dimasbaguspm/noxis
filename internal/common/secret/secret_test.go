package secret

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
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
	k, err := New(newKey(t))
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
	plain, err := k.Open("ref-a", sealed)
	if err != nil || string(plain) != "s3cret" {
		t.Fatalf("Open = %q, %v", plain, err)
	}
	if _, err := k.Open("ref-b", sealed); !errors.Is(err, ErrOpen) {
		t.Fatalf("Open with a different aad = %v, want ErrOpen", err)
	}
}

// There is one key. A row sealed with another one refuses, and the refusal says what to do: this is
// the honest cost of dropping SECRET_KEY_PREVIOUS.
func TestAnotherKeyRefuses(t *testing.T) {
	before, err := mustNew(t, newKey(t)).Seal("ref", []byte("pw"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := mustNew(t, newKey(t)).Open("ref", before); !errors.Is(err, ErrOpen) {
		t.Fatalf("Open with another key = %v, want ErrOpen", err)
	} else if !strings.Contains(err.Error(), "re-enter the target password") {
		t.Fatalf("refusal = %q, want the instruction", err)
	}
}

func TestNoKey(t *testing.T) {
	k, err := New(nil)
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
	k := mustNew(t, newKey(t))
	for _, sealed := range []string{"", "s3cret", "v2:AAAA", "v1:not-base64!!"} {
		if _, err := k.Open("ref", sealed); err == nil {
			t.Fatalf("Open(%q) accepted garbage", sealed)
		}
	}
}

func TestNewRejectsWrongLength(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
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

func mustNew(t *testing.T, key []byte) *Keyring {
	t.Helper()
	k, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return k
}

// A rotation is a deployment decision: the old key still opens what it sealed, and everything sealed
// afterwards uses the new one. Without that, replacing a key would mean re-entering every password at
// once, in the middle of an incident, or losing them.
func TestRotationOpensWithTheOldKeyAndSealsWithTheNew(t *testing.T) {
	oldKey := bytes.Repeat([]byte{'a'}, KeyLength)
	newKey := bytes.Repeat([]byte{'b'}, KeyLength)

	before, err := New(oldKey)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sealed, err := before.Seal("target-ref", []byte("hunter2"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	rotated, err := NewWithPrevious(newKey, oldKey)
	if err != nil {
		t.Fatalf("NewWithPrevious: %v", err)
	}
	plain, err := rotated.Open("target-ref", sealed)
	if err != nil {
		t.Fatalf("Open with the previous key: %v", err)
	}
	if string(plain) != "hunter2" {
		t.Fatalf("plain = %q, want the password back", plain)
	}

	// What the rotated keyring writes opens with itself and not with the old key alone.
	resealed, err := rotated.Seal("target-ref", plain)
	if err != nil {
		t.Fatalf("Seal after rotation: %v", err)
	}
	if _, err := before.Open("target-ref", resealed); err == nil {
		t.Fatal("a value sealed after the rotation opened with the old key alone")
	}
	if _, err := rotated.Open("target-ref", resealed); err != nil {
		t.Fatalf("Open after rotation: %v", err)
	}
}

// A row sealed by a key nobody holds any more says what to do, and names the variable that fixes it.
func TestOpenNamesTheMissingKey(t *testing.T) {
	other, err := New(bytes.Repeat([]byte{'z'}, KeyLength))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sealed, err := other.Seal("target-ref", []byte("hunter2"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	mine, err := New(bytes.Repeat([]byte{'q'}, KeyLength))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = mine.Open("target-ref", sealed)
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("error = %v, want ErrOpen", err)
	}
	if !strings.Contains(err.Error(), "SECRET_KEY_OLD") {
		t.Fatalf("error = %q, want it to name SECRET_KEY_OLD", err)
	}
}
