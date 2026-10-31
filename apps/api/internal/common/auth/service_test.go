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
	bySubject         map[string]*entity.User
	tokens            map[string]fakeToken
	getUserByEmailErr error
}

type fakeToken struct {
	subject string
	expires int64
}

// rootEmail is the env account's identifier in these tests.
const rootEmail = "root@example.com"

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

func (f *fakeUsers) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	if f.getUserByEmailErr != nil {
		return nil, f.getUserByEmailErr
	}
	for _, u := range f.bySubject {
		if u.Email == email && email != "" {
			cp := *u
			return &cp, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *fakeUsers) GetUserByLogin(_ context.Context, login string) (*entity.User, error) {
	for _, u := range f.bySubject {
		if login != "" && u.Password != "" && (u.Email == login || u.Username == login) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *fakeUsers) EnsureSubject(_ context.Context, subject, email string) error {
	if _, ok := f.bySubject[subject]; !ok {
		f.bySubject[subject] = &entity.User{Subject: subject, Email: email, Role: entity.RoleMember, Status: entity.StatusActive}
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

func (f *fakeUsers) SetUserStatus(_ context.Context, subject, status string) error {
	if u, ok := f.bySubject[subject]; ok {
		u.Status = status
	}
	return nil
}

func (f *fakeUsers) SetUserRole(_ context.Context, subject, role string) error {
	if u, ok := f.bySubject[subject]; ok {
		u.Role = role
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
		switch {
		case s == subject:
			u.Role = entity.RoleSuper
		case u.Role == entity.RoleSuper:
			u.Role = entity.RoleMember
		}
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
	if !ok || !u.Active() {
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
	if err := svc.EnsureRoot(context.Background(), rootEmail, "supersecret"); err != nil {
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
	if err := svc.EnsureRoot(ctx, rootEmail, "short"); err == nil {
		t.Fatal("a too-short root password must fail")
	}
	if err := svc.EnsureRoot(ctx, rootEmail, "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	// a later boot with different env must not overwrite the stored password
	if err := svc.EnsureRoot(ctx, rootEmail, "different-long"); err != nil {
		t.Fatalf("second seed must be a no-op: %v", err)
	}
	if _, _, err := svc.Login(ctx, rootEmail, "supersecret"); err != nil {
		t.Fatalf("original password must still work: %v", err)
	}
}

func TestEnsureRootIsSoleSuper(t *testing.T) {
	repo := newFakeUsers()
	svc := NewService(repo, nil)
	ctx := context.Background()

	if err := svc.EnsureRoot(ctx, rootEmail, "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	// a second super, as an older build might have stored, is cleared on boot
	if err := repo.SaveUser(ctx, entity.User{Subject: "local:bob@example.com", Email: "bob@example.com", Role: entity.RoleSuper}); err != nil {
		t.Fatalf("seed bob: %v", err)
	}
	if err := svc.EnsureRoot(ctx, rootEmail, ""); err != nil {
		t.Fatalf("reboot: %v", err)
	}
	if u, _ := repo.GetUser(ctx, "local:bob@example.com"); u.IsSuper() {
		t.Fatal("only the env account may be super")
	}
	if u, _ := repo.GetUser(ctx, entity.LocalSubject(rootEmail)); !u.IsSuper() {
		t.Fatal("the env account must be super")
	}
	if err := svc.EnsureRoot(ctx, "ghost@example.com", ""); err == nil {
		t.Fatal("an unknown env account must fail closed")
	}
}

func TestLoginVerifyLogout(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, rootEmail, "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := svc.Verify(ctx, token)
	if err != nil || id.Subject != entity.LocalSubject(rootEmail) || !id.IsSuper() {
		t.Fatalf("verify: %+v err=%v", id, err)
	}
	if _, err := svc.Verify(ctx, "not-a-token"); err == nil {
		t.Fatal("an unknown token must fail")
	}
	if err := svc.Logout(ctx, entity.LocalSubject(rootEmail)); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("token must be dead after logout")
	}
}

func TestLoginRejects(t *testing.T) {
	svc, repo := seeded(t)
	ctx := context.Background()

	if _, _, err := svc.Login(ctx, rootEmail, "wrong-password"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, _, err := svc.Login(ctx, "ghost@example.com", "whatever-long"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("unknown user: %v", err)
	}
	u, _ := repo.GetUser(ctx, entity.LocalSubject(rootEmail))
	u.Status = entity.StatusBanned
	_ = repo.SaveUser(ctx, *u)
	if _, _, err := svc.Login(ctx, rootEmail, "supersecret"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("disabled user: %v", err)
	}
}

func TestChangePasswordRotatesToken(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, rootEmail, "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.ChangePassword(ctx, entity.LocalSubject(rootEmail), "wrong", "another-long"); !errors.Is(err, constants.ErrInvalidCredentials) {
		t.Fatalf("old password check: %v", err)
	}
	if err := svc.ChangePassword(ctx, entity.LocalSubject(rootEmail), "supersecret", "short"); err == nil {
		t.Fatal("a too-short new password must fail")
	}
	if err := svc.ChangePassword(ctx, entity.LocalSubject(rootEmail), "supersecret", "another-long"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("the old token must be dead after a password change")
	}
	if _, _, err := svc.Login(ctx, rootEmail, "another-long"); err != nil {
		t.Fatalf("new password must work: %v", err)
	}
}

func TestTokenExpires(t *testing.T) {
	svc, _ := seeded(t)
	ctx := context.Background()
	base := time.Now()
	svc.now = func() time.Time { return base }

	token, _, err := svc.Login(ctx, rootEmail, "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	svc.now = func() time.Time { return base.Add(TokenTTL + time.Minute) }
	if _, err := svc.Verify(ctx, token); err == nil {
		t.Fatal("an expired token must fail")
	}
}

// An OIDC identity is recorded as a reference row the first time it is seen, so grants and the user directory can point at it.
func TestVerifyRecordsOIDCSubject(t *testing.T) {
	repo := newFakeUsers()
	oidc := fakeVerifier{identity: entity.Identity{
		Subject: "oidc|42", Name: "OIDC User", Email: "oidc@example.com",
		Groups: []string{"dev"}, Role: entity.RoleSuper,
	}}
	svc := NewService(repo, oidc)
	ctx := context.Background()

	id, err := svc.Verify(ctx, "an-oidc-token")
	if err != nil || id.Subject != "oidc|42" || id.Role != entity.RoleMember || id.Name != "OIDC User" || len(id.Groups) != 1 || id.Groups[0] != "dev" {
		t.Fatalf("verify: %+v err=%v", id, err)
	}
	if _, err := repo.GetUser(ctx, "oidc|42"); err != nil {
		t.Fatalf("the OIDC subject must be recorded: %v", err)
	}
}

// A previously recorded unmatched OIDC subject cannot authenticate after it is banned.
func TestVerifyRejectsBannedUnmatchedOIDCSubject(t *testing.T) {
	repo := newFakeUsers()
	if err := repo.EnsureSubject(context.Background(), "oidc|banned", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetUserStatus(context.Background(), "oidc|banned", entity.StatusBanned); err != nil {
		t.Fatal(err)
	}
	svc := NewService(repo, fakeVerifier{identity: entity.Identity{Subject: "oidc|banned"}})
	if _, err := svc.Verify(context.Background(), "an-oidc-token"); !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("banned unmatched OIDC user error = %v, want forbidden", err)
	}
}

func TestVerifyPropagatesVerifiedEmailLookupFailure(t *testing.T) {
	repo := newFakeUsers()
	repo.getUserByEmailErr = errors.New("sqlite unavailable")
	svc := NewService(repo, fakeVerifier{identity: entity.Identity{
		Subject: "oidc|42", Email: "oidc@example.com", EmailVerified: true,
	}})
	if _, err := svc.Verify(context.Background(), "an-oidc-token"); err == nil || !errors.Is(err, repo.getUserByEmailErr) {
		t.Fatalf("email lookup error = %v, want propagated storage error", err)
	}
	if _, err := repo.GetUser(context.Background(), "oidc|42"); !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("lookup failure created OIDC user: err=%v", err)
	}
}

// With no OIDC verifier, a non-local token is rejected.
func TestServiceLocalOnlyRejects(t *testing.T) {
	svc, _ := seeded(t)
	if _, err := svc.Verify(context.Background(), "oidc-looking-token"); err == nil {
		t.Fatal("with no OIDC verifier, a non-local token must be rejected")
	}
}

// An OIDC login whose verified email matches a local account signs in as that account: same subject, same role. An unverified claim never blends.
func TestVerifyBlendsVerifiedOIDCEmail(t *testing.T) {
	repo := newFakeUsers()
	ctx := context.Background()
	svc := NewService(repo, fakeVerifier{identity: entity.Identity{Subject: "oidc|root", Email: rootEmail, Name: "Root", EmailVerified: true}})
	if err := svc.EnsureRoot(ctx, rootEmail, "supersecret"); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	id, err := svc.Verify(ctx, "an-oidc-token")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id.Subject != entity.LocalSubject(rootEmail) || !id.IsSuper() {
		t.Fatalf("blend: %+v", id)
	}
	if _, err := repo.GetUser(ctx, "oidc|root"); err == nil {
		t.Fatal("a blended identity must not create a second account")
	}

	// a banned local account rejects the blended login instead of creating one
	u, _ := repo.GetUser(ctx, entity.LocalSubject(rootEmail))
	u.Status = entity.StatusBanned
	_ = repo.SaveUser(ctx, *u)
	if _, err := svc.Verify(ctx, "an-oidc-token"); !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("banned blend: %v", err)
	}
}

// An unverified or absent email_verified claim must not join a local account, no matter which email it carries: it is recorded as a plain member reference.
func TestVerifyRejectsUnverifiedOIDCEmail(t *testing.T) {
	repo := newFakeUsers()
	ctx := context.Background()
	if err := svcRoot(ctx, repo); err != nil {
		t.Fatal(err)
	}
	svc := NewService(repo, fakeVerifier{identity: entity.Identity{Subject: "oidc|attacker", Email: rootEmail, EmailVerified: false}})
	if _, err := svc.Verify(ctx, "an-oidc-token"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	id, err := svc.Verify(ctx, "an-oidc-token")
	if err != nil || id.Subject != "oidc|attacker" || id.IsAdmin() {
		t.Fatalf("an unverified claim must stay a plain member reference: %+v err=%v", id, err)
	}
	// the local account was never touched
	if u, _ := repo.GetUser(ctx, entity.LocalSubject(rootEmail)); u.Email != rootEmail || u.Subject == "oidc|attacker" {
		t.Fatalf("local account: %+v", u)
	}
}

func svcRoot(ctx context.Context, repo *fakeUsers) error {
	svc := NewService(repo, nil)
	return svc.EnsureRoot(ctx, rootEmail, "supersecret")
}
