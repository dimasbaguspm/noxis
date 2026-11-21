// Package secret seals target passwords with AES-256-GCM, so the state file on its own
// is not a list of database credentials.
//
// The sealed form is `v1:<base64(nonce|ciphertext|tag)>`. The version prefix is what
// lets a later scheme read old rows instead of refusing them. The additional
// authenticated data is the target's ref, so a sealed password cannot be moved to
// another row and still open.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// KeyLength is the AES-256 key size, in bytes.
const KeyLength = 32

const prefix = "v1:"

// Errors callers compare with errors.Is.
var (
	// ErrNoKey means no SECRET_KEY is configured, so nothing can be sealed or opened.
	ErrNoKey = errors.New("secret: no key configured")
	// ErrOpen means the envelope is malformed, was sealed with an unknown key, or its
	// additional data does not match the row it was read from.
	ErrOpen = errors.New("secret: cannot open")
)

// ParseKey reads a 32-byte key from 64 hex characters or from base64.
func ParseKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrNoKey
	}
	if len(s) == KeyLength*2 {
		if b, err := hex.DecodeString(s); err == nil {
			return b, nil
		}
	}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == KeyLength {
			return b, nil
		}
	}
	return nil, fmt.Errorf("secret: key is not %d bytes as hex or base64", KeyLength)
}

// Keyring seals with the current key and opens with the current key or any previous one. Rotation is a
// deployment decision — add the old key with SECRET_KEY_OLD, re-save the passwords, then drop it — so a
// row sealed before a rotation still opens while the operator works through the targets, and every value
// written after it is sealed by the key they intend to keep.
type Keyring struct {
	// keys[0] seals; all of them open, current first.
	keys [][]byte
}

// New builds a keyring with one key. An empty key is allowed — the server runs, and a target that
// carries a password cannot be registered until a key is set.
func New(key []byte) (*Keyring, error) { return NewWithPrevious(key) }

// NewWithPrevious builds a keyring that seals with current and also opens what previous sealed.
func NewWithPrevious(current []byte, previous ...[]byte) (*Keyring, error) {
	keys := make([][]byte, 0, 1+len(previous))
	for _, key := range append([][]byte{current}, previous...) {
		if len(key) == 0 {
			continue
		}
		if len(key) != KeyLength {
			return nil, fmt.Errorf("secret: key is %d bytes, want %d", len(key), KeyLength)
		}
		keys = append(keys, key)
	}
	return &Keyring{keys: keys}, nil
}

// Enabled reports whether anything can be sealed.
func (k *Keyring) Enabled() bool { return k != nil && len(k.keys) > 0 }

// Seal encrypts plain with the current key, binding it to aad.
func (k *Keyring) Seal(aad string, plain []byte) (string, error) {
	if !k.Enabled() {
		return "", ErrNoKey
	}
	gcm, err := newGCM(k.keys[0])
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secret: nonce: %w", err)
	}
	return prefix + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plain, []byte(aad))), nil
}

// Open decrypts a sealed value with the current key, then with every previous one.
func (k *Keyring) Open(aad, sealed string) ([]byte, error) {
	if !k.Enabled() {
		return nil, ErrNoKey
	}
	rest, ok := strings.CutPrefix(sealed, prefix)
	if !ok {
		return nil, fmt.Errorf("%w: unknown envelope", ErrOpen)
	}
	raw, err := base64.StdEncoding.DecodeString(rest)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpen, err)
	}
	for _, key := range k.keys {
		gcm, err := newGCM(key)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrOpen, err)
		}
		if len(raw) < gcm.NonceSize() {
			return nil, ErrOpen
		}
		plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(aad))
		if err == nil {
			return plain, nil
		}
	}
	return nil, fmt.Errorf("%w: the sealed value was written with a key this deployment does not hold — set SECRET_KEY_OLD to the key that wrote it, or re-enter the target password", ErrOpen)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: cipher: %w", err)
	}
	return cipher.NewGCM(block)
}
