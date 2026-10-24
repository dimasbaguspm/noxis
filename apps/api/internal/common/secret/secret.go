// Package secret encrypts stored credentials reversibly. Passwords cannot be hashed — noxis must send them to the database — so they are encrypted at rest instead, with a key that never lives in the same file.
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

// prefix marks an encrypted value, so plaintext rows from before the key existed remain readable and can be migrated.
const prefix = "enc:v1:"

// Cipher uses AES-256-GCM; a nil cipher passes stored values through unchanged.
type Cipher struct {
	aead cipher.AEAD
}

// New builds a cipher from a 32-byte key, given as 64 hex characters or 32 raw bytes.
func New(key string) (*Cipher, error) {
	if key == "" {
		return nil, nil
	}
	raw, err := decodeKey(key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func decodeKey(key string) ([]byte, error) {
	if len(key) == 64 {
		if raw, err := hex.DecodeString(key); err == nil {
			return raw, nil
		}
	}
	if len(key) == 32 {
		return []byte(key), nil
	}
	return nil, fmt.Errorf("secret key must be 64 hex characters or 32 raw bytes, got %d characters", len(key))
}

// Encrypt seals non-empty secrets; nil cipher or empty input passes through unchanged.
func (c *Cipher) Encrypt(plain string) (string, error) {
	if c == nil || plain == "" {
		return plain, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secret nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens encrypted secrets, passes legacy plaintext through, and rejects ciphertext without a key.
func (c *Cipher) Decrypt(stored string) (string, error) {
	if !Encrypted(stored) {
		return stored, nil
	}
	if c == nil {
		return "", errors.New("stored secret is encrypted but NOXIS_SECRET_KEY is not set")
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("decode stored secret: %w", err)
	}
	n := c.aead.NonceSize()
	if len(raw) < n {
		return "", errors.New("stored secret is truncated")
	}
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", errors.New("stored secret cannot be decrypted (wrong NOXIS_SECRET_KEY?)")
	}
	return string(plain), nil
}

// Encrypted reports whether a stored value is encrypted.
func Encrypted(stored string) bool { return strings.HasPrefix(stored, prefix) }
