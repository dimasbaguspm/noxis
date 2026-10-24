package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// newStore opens a throwaway store for the account tests.
func newStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

// The email is the account identifier: the login identifier resolves through it, and a legacy username still matches until an email takes over.
func TestUserLookups(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	root := entity.User{
		Subject: entity.LocalSubject("root@example.com"), Email: "root@example.com", Username: "root@example.com",
		Password: "hash", Role: entity.RoleSuper, Status: entity.StatusActive,
	}
	if err := store.SaveUser(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}

	if u, err := store.GetUserByLogin(ctx, "root@example.com"); err != nil || u.Subject != root.Subject {
		t.Fatalf("login by email: %+v err=%v", u, err)
	}
	if _, err := store.GetUserByLogin(ctx, ""); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("an empty login must not match: %v", err)
	}
	if _, err := store.GetUserByLogin(ctx, "nobody@example.com"); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("an unknown login must not match: %v", err)
	}
	if _, err := store.GetUserByLogin(ctx, "nobody"); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("an unknown username must not match: %v", err)
	}
	// a second super is demoted on the next boot
	if err := store.SaveUser(ctx, entity.User{Subject: "local:bob@example.com", Email: "bob@example.com", Username: "bob@example.com", Role: entity.RoleSuper, Status: entity.StatusActive}); err != nil {
		t.Fatalf("save bob: %v", err)
	}
	if err := store.SetSoleSuper(ctx, root.Subject); err != nil {
		t.Fatalf("sole super: %v", err)
	}
	if u, _ := store.GetUser(ctx, "local:bob@example.com"); u.IsSuper() {
		t.Fatal("the second super must be demoted")
	}
	if u, _ := store.GetUser(ctx, root.Subject); !u.IsSuper() {
		t.Fatal("the env account must stay super")
	}
}

// EnsureSubject records an OIDC row once, carrying its email but no password.
func TestEnsureSubjectIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	for i := 0; i < 2; i++ {
		if err := store.EnsureSubject(ctx, "oidc|42", "carol@example.com"); err != nil {
			t.Fatalf("ensure %d: %v", i, err)
		}
	}
	u, err := store.GetUser(ctx, "oidc|42")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if u.Email != "carol@example.com" || u.Password != "" || u.Role != entity.RoleMember || !u.Active() {
		t.Fatalf("row: %+v", u)
	}
	// an OIDC row must not sign in locally: no password to check
	if _, err := store.GetUserByLogin(ctx, "carol@example.com"); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("an OIDC row has no password and must not log in locally: %v", err)
	}
}
