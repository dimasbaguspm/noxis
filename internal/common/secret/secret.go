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

// Keyring seals with the one key the deployment was given. There is no previous key: a state file
// sealed with another one refuses, and the refusal says to re-enter the password rather than
// pretending a rotation is transparent.
type Keyring struct {
	key []byte
}

// New builds a keyring. An empty key is allowed — the server runs, and a target that carries a
// password cannot be registered until a key is set. A key of the wrong length is an error rather
// than a silent truncation.
func New(key []byte) (*Keyring, error) {
	if len(key) == 0 {
		return &Keyring{}, nil
	}
	if len(key) != KeyLength {
		return nil, fmt.Errorf("secret: key is %d bytes, want %d", len(key), KeyLength)
	}
	return &Keyring{key: key}, nil
}

// Enabled reports whether anything can be sealed.
func (k *Keyring) Enabled() bool { return k != nil && len(k.key) > 0 }

// Seal encrypts plain with the current key, binding it to aad.
func (k *Keyring) Seal(aad string, plain []byte) (string, error) {
	if !k.Enabled() {
		return "", ErrNoKey
	}
	gcm, err := newGCM(k.key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secret: nonce: %w", err)
	}
	return prefix + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plain, []byte(aad))), nil
}

// Open decrypts a sealed value with the key this process was given.
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
	gcm, err := newGCM(k.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpen, err)
	}
	if len(raw) < gcm.NonceSize() {
		return nil, ErrOpen
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("%w: the sealed value was written with another key, so it cannot be opened — re-enter the target password", ErrOpen)
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: cipher: %w", err)
	}
	return cipher.NewGCM(block)
}
