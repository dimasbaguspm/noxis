package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// fakeUsers is an in-memory UserRepository for the service tests.
type fakeUsers struct {
	bySubject map[string]*entity.User
	tokens    map[string]fakeToken
}

type fakeToken struct {
	subject string
	expires int64
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{bySubject: map[string]*entity.User{}, tokens: map[string]fakeToken{}}
}

func (f *fakeUsers) CountUsers(context.Context) (int, error) { return len(f.bySubject), nil }

func (f *fakeUsers) ListUsers(context.Context) ([]entity.User, error) {
	out := make([]entity.User, 0, len(f.bySubject))
	for _, u := range f.bySubject {
		out = append(out, *u)
	}
	return out, nil
}

func (f *fakeUsers) GetUser(_ context.Context, subject string) (*entity.User, error) {
	u, ok := f.bySubject[subject]
	if !ok {
		return nil, constants.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUsers) GetUserByUsername(_ context.Context, username string) (*entity.User, error) {
	for _, u := range f.bySubject {
		if u.Username == username && username != "" {
			cp := *u
			return &cp, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *fakeUsers) EnsureSubject(_ context.Context, subject string) error {
	if _, ok := f.bySubject[subject]; !ok {
		f.bySubject[subject] = &entity.User{Subject: subject}
	}
	return nil
}

func (f *fakeUsers) SaveUser(_ context.Context, u entity.User) error {
	cp := u
	f.bySubject[u.Subject] = &cp
	return nil
}

func (f *fakeUsers) SetUserPassword(_ context.Context, subject, hash string) error {
	u, ok := f.bySubject[subject]
	if !ok {
		return constants.ErrNotFound
	}
	u.Password = hash
	return nil
}

func (f *fakeUsers) SetUserToken(_ context.Context, subject, tokenHash string, expiresAt int64) error {
	if _, ok := f.bySubject[subject]; !ok {
		return constants.ErrNotFound
	}
	for h, t := range f.tokens { // single active token
		if t.subject == subject {
			delete(f.tokens, h)
		}
	}
	f.tokens[tokenHash] = fakeToken{subject: subject, expires: expiresAt}
	return nil
}

func (f *fakeUsers) SetUserDisabled(_ context.Context, subject string, disabled bool) error {
	if u, ok := f.bySubject[subject]; ok {
		u.Disabled = disabled
	}
	return nil
}

func (f *fakeUsers) SetLastLogin(_ context.Context, subject string, at int64) error {
	if u, ok := f.bySubject[subject]; ok {
		u.LastLoginAt = at
	}
	return nil
}

func (f *fakeUsers) SetSoleSuper(_ context.Context, subject string) error {
	for s, u := range f.bySubject {
		u.Super = s == subject
	}
	return nil
}

func (f *fakeUsers) ClearUserToken(_ context.Context, subject string) error {
	for h, t := range f.tokens {
		if t.subject == subject {
			delete(f.tokens, h)
		}
	}
	return nil
}

func (f *fakeUsers) UserByToken(_ context.Context, tokenHash string, now int64) (*entity.User, error) {
	t, ok := f.tokens[tokenHash]
	if !ok || t.expires <= now {
		return nil, constants.ErrNotFound
	}
	u, ok := f.bySubject[t.subject]
	if !ok || u.Disabled {
		return nil, constants.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

// fakeVerifier stands in for the OIDC verifier.
type fakeVerifier struct {
	identity entity.Identity
	err      error
}

func (f fakeVerifier) Verify(context.Context, string) (entity.Identity, error) {
	return f.identity, f.err
}

func seeded(t *testing.T) (*Service, *fakeUsers) {
	t.Helper()
	repo := newFakeUsers()
	svc := NewService(repo, nil)
	if err := svc.EnsureRoot(context.Background(), "root", "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	return svc, repo
}

func TestEnsureRootCreatesOnce(t *testing.T) {
	repo := newFakeUsers()
	svc := NewService(repo, nil)
	ctx := context.Background()

	if err := svc.EnsureRoot(ctx, "", ""); err == nil {
		t.Fatal("empty store with no env must fail")
	}
	if err := svc.EnsureRoot(ctx, "root", "short"); err == nil {
		t.Fatal("a too-short root password must fail")
	}
	if err := svc.EnsureRoot(ctx, "root", "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	// a later boot with different env must not overwrite the stored password
	if err := svc.EnsureRoot(ctx, "root", "different-long"); err != nil {
		t.Fatalf("second seed must be a no-op: %v", err)
	}
	if _, _, err := svc.Login(ctx, "root", "supersecret"); err != nil {
		t.Fatalf("original password must still work: %v", err)
	}
}

func TestEnsureRootIsSoleSuper(t *testing.T) {
	repo := newFakeUsers()
	svc := NewService(repo, nil)
	ctx := context.Background()

	if err := svc.EnsureRoot(ctx, "root", "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	// a second super, as an older build might have stored, is cleared on boot
	if err := repo.SaveUser(ctx, entity.User{Subject: "local:bob", Username: "bob", Super: true}); err != nil {
		t.Fatalf("seed bob: %v", err)
	}
	if err := svc.EnsureRoot(ctx, "root", ""); err != nil {
		t.Fatalf("reboot: %v", err)
	}
	if u, _ := repo.GetUser(ctx, "local:bob"); u.Super {
		t.Fatal("only the env account may be super")
	}
	if u, _ := repo.GetUser(ctx, "local:root"); !u.Super {
		t.Fatal("the env account must be super")
	}
	if err := svc.EnsureRoot(ctx, "ghost", ""); err == nil {
		t.Fatal("an unknown env account must fail closed")
	}
}

func TestLoginVerifyLogout(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "root", "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := svc.Verify(ctx, token)
	if err != nil || id.Subject != "local:root" || !id.Super {
		t.Fatalf("verify: %+v err=%v", id, err)
	}
	if _, err := svc.Verify(ctx, "not-a-token"); err == nil {
		t.Fatal("an unknown token must fail")
	}
	if err := svc.Logout(ctx, "local:root"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("token must be dead after logout")
	}
}

func TestLoginRejects(t *testing.T) {
	svc, repo := seeded(t)
	ctx := context.Background()

	if _, _, err := svc.Login(ctx, "root", "wrong-password"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, _, err := svc.Login(ctx, "ghost", "whatever-long"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("unknown user: %v", err)
	}
	u, _ := repo.GetUser(ctx, "local:root")
	u.Disabled = true
	_ = repo.SaveUser(ctx, *u)
	if _, _, err := svc.Login(ctx, "root", "supersecret"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("disabled user: %v", err)
	}
}

func TestChangePasswordRotatesToken(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "root", "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.ChangePassword(ctx, "local:root", "wrong", "another-long"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("old password check: %v", err)
	}
	if err := svc.ChangePassword(ctx, "local:root", "supersecret", "short"); err == nil {
		t.Fatal("a too-short new password must fail")
	}
	if err := svc.ChangePassword(ctx, "local:root", "supersecret", "another-long"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("the old token must be dead after a password change")
	}
	if _, _, err := svc.Login(ctx, "root", "another-long"); err != nil {
		t.Fatalf("new password must work: %v", err)
	}
}

func TestTokenExpires(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()
	base := time.Now()
	svc.now = func() time.Time { return base }

	token, _, err := svc.Login(ctx, "root", "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	svc.now = func() time.Time { return base.Add(TokenTTL + time.Minute) }
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("an expired token must fail")
	}
}

// An OIDC identity is recorded as a reference row the first time it is seen, so
// grants and the user directory can point at it.
func TestVerifyRecordsOIDCSubject(t *testing.T) {
	repo := newFakeUsers()
	oidc := fakeVerifier{identity: entity.Identity{Subject: "oidc|42", Groups: []string{"dev"}}}
	svc := NewService(repo, oidc)
	ctx := context.Background()

	id, err := svc.Verify(ctx, "an-oidc-token")
	if err != nil || id.Subject != "oidc|42" {
		t.Fatalf("verify: %+v err=%v", id, err)
	}
	if _, err := repo.GetUser(ctx, "oidc|42"); err != nil {
		t.Fatalf("the OIDC subject must be recorded: %v", err)
	}
}

// With no OIDC verifier, a non-local token is rejected.
func TestServiceLocalOnlyRejects(t *testing.T) {
	svc, _ := seeded(t)
	if _, err := svc.Verify(context.Background(), "oidc-looking-token"); err == nil {
		t.Fatal("with no OIDC verifier, a non-local token must be rejected")
	}
}
