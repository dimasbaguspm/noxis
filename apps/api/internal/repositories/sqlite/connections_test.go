package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"noxis/internal/common/secret"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const testSecretKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

// With a key the store must never persist plaintext: stored passwords are sealed on open, and new writes are sealed on save.
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

func TestDeleteConnectionRetainsChangeHistory(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Save(ctx, entity.Connection{Name: "demo", Driver: "postgres", Host: "db", Port: 5432, User: "postgres", Database: "demo"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetPermission(ctx, entity.Permission{Connection: "demo", Member: "root", MemberType: entity.MemberUser, Level: entity.LevelRead}); err != nil {
		t.Fatal(err)
	}
	change := entity.Change{ID: "change-1", Connection: "demo", Notes: "change", Query: "INSERT", Rollback: "DELETE", Kind: constants.KindDML, Status: constants.ChangeValidated, CreatedAt: 1}
	if err := store.SaveChange(ctx, change); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "demo"); err != nil {
		t.Fatalf("delete connection: %v", err)
	}
	if _, err := store.Get(ctx, "demo"); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("deleted connection error = %v, want not found", err)
	}
	if permissions, err := store.ListPermissions(ctx, "demo"); err != nil || len(permissions) != 0 {
		t.Fatalf("permissions after delete = %+v, error = %v", permissions, err)
	}
	if got, err := store.GetChange(ctx, change.ID); err != nil || got.Connection != "demo" {
		t.Fatalf("change history after delete = %+v, error = %v", got, err)
	}
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
