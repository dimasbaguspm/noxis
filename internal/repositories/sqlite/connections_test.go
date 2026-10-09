package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"noxis/internal/common/secret"
	"noxis/internal/entity"
)

const testSecretKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

// With a key the store must never persist plaintext: a legacy row is sealed on
// open, and new writes are sealed on save.
func TestPasswordsEncryptedAtRest(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "noxis.db")
	conn := entity.Connection{
		Name: "demo", Driver: "postgres", Host: "db", Port: 5432,
		User: "postgres", Password: "s3cret", Database: "demo",
	}

	plain, err := Open(ctx, path, "")
	if err != nil {
		t.Fatalf("open plain: %v", err)
	}
	if err := plain.Save(ctx, conn); err != nil {
		t.Fatalf("save plain: %v", err)
	}
	plain.Close()

	store, err := Open(ctx, path, testSecretKey)
	if err != nil {
		t.Fatalf("open encrypted: %v", err)
	}
	defer store.Close()

	got, err := store.Get(ctx, "demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Password != "s3cret" {
		t.Fatalf("password = %q, want s3cret", got.Password)
	}
	assertCiphertext(t, store, "demo")

	conn.Name, conn.Password = "other", "top"
	if err := store.Save(ctx, conn); err != nil {
		t.Fatalf("save encrypted: %v", err)
	}
	assertCiphertext(t, store, "other")
}

func assertCiphertext(t *testing.T, store *Store, name string) {
	t.Helper()
	var raw string
	if err := store.db.QueryRow(`SELECT password FROM connections WHERE name = ?`, name).Scan(&raw); err != nil {
		t.Fatalf("read raw password: %v", err)
	}
	if !secret.Encrypted(raw) || strings.Contains(raw, "s3cret") || strings.Contains(raw, "top") {
		t.Fatalf("stored password is not ciphertext: %q", raw)
	}
}
