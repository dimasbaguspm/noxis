package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const (
	// TokenTTL bounds how long a local login survives.
	TokenTTL = 12 * time.Hour
	// MinPasswordLength is enforced for new and changed passwords.
	MinPasswordLength = 8
)

// dummyHash equalizes login timing when the account is unknown. Generating it
// once at boot costs a single bcrypt pass.
var dummyHash = func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("noxis-timing-equalizer"), bcrypt.DefaultCost)
	return h
}()

// Service owns local accounts and is the process verifier: it accepts a local
// session token, otherwise the configured OIDC verifier.
type Service struct {
	users interfaces.UserRepository
	oidc  interfaces.Verifier // nil when OIDC is not configured
	now   func() time.Time
}

// Satisfier.
var _ interfaces.Verifier = (*Service)(nil)

// NewService builds the auth service. The OIDC verifier is optional; pass a nil
// interface (not a typed-nil) to disable it.
func NewService(users interfaces.UserRepository, oidc interfaces.Verifier) *Service {
	return &Service{users: users, oidc: oidc, now: time.Now}
}

// Verify resolves a bearer token: a local session first (cheap, indexed), then
// OIDC when configured. Every OIDC identity is recorded as a reference row the
// first time it is seen. Anything else is rejected.
func (s *Service) Verify(ctx context.Context, rawToken string) (entity.Identity, error) {
	if rawToken == "" {
		return entity.Identity{}, constants.ErrInvalidCredentials
	}
	if u, err := s.users.UserByToken(ctx, TokenHash(rawToken), s.now().Unix()); err == nil {
		return u.Identity(), nil
	}
	if s.oidc == nil {
		return entity.Identity{}, constants.ErrInvalidCredentials
	}
	id, err := s.oidc.Verify(ctx, rawToken)
	if err != nil {
		return entity.Identity{}, err
	}
	if err := s.users.EnsureSubject(ctx, id.Subject); err != nil {
		return entity.Identity{}, err
	}
	return id, nil
}

// EnsureRoot makes the env account the single super admin. On a cold install it
// creates the account; on an existing store it promotes that account and clears
// any other super, so the env is always the source of the one global role. It
// fails closed when no email is configured.
func (s *Service) EnsureRoot(ctx context.Context, email, password string) error {
	email = normalizeEmail(email)
	if email == "" {
		return errors.New("NOXIS_ADMIN_EMAIL is required: it names the single super admin")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("NOXIS_ADMIN_EMAIL %q is not an email address: %w", email, err)
	}
	subject := entity.LocalSubject(email)
	_, err := s.users.GetUser(ctx, subject)
	switch {
	case errors.Is(err, constants.ErrNotFound):
		if password == "" {
			return fmt.Errorf("no account for %q; set NOXIS_ADMIN_PASSWORD to create the super admin", email)
		}
		if err := checkPassword(password); err != nil {
			return err
		}
		hash, err := HashPassword(password)
		if err != nil {
			return err
		}
		if err := s.users.SaveUser(ctx, entity.User{
			Subject:  subject,
			Email:    email,
			Username: email,
			Password: hash,
			Role:     entity.RoleSuper,
		}); err != nil {
			return err
		}
	case err != nil:
		return err
	}
	return s.users.SetSoleSuper(ctx, subject)
}

// Login verifies a password and issues the account's single active token. The
// identifier is the account's email (or the username it was created with).
func (s *Service) Login(ctx context.Context, login, password string) (string, entity.User, error) {
	u, err := s.users.GetUserByLogin(ctx, normalizeEmail(login))
	if err != nil {
		// pay the bcrypt cost anyway: timing must not reveal a known account
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", entity.User{}, constants.ErrInvalidCredentials
	}
	if u.Disabled || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return "", entity.User{}, constants.ErrInvalidCredentials
	}
	token, hash, err := newToken()
	if err != nil {
		return "", entity.User{}, err
	}
	now := s.now()
	if err := s.users.SetUserToken(ctx, u.Subject, hash, now.Add(TokenTTL).Unix()); err != nil {
		return "", entity.User{}, err
	}
	if err := s.users.SetLastLogin(ctx, u.Subject, now.Unix()); err != nil {
		return "", entity.User{}, err
	}
	u.LastLoginAt = now.Unix()
	return token, *u, nil
}

// Logout clears the account's active token.
func (s *Service) Logout(ctx context.Context, subject string) error {
	return s.users.ClearUserToken(ctx, subject)
}

// ChangePassword verifies the old password, stores the new hash, and rotates the
// token so a stolen session does not survive the change. Only local accounts
// have a password.
func (s *Service) ChangePassword(ctx context.Context, subject, oldPassword, newPassword string) error {
	u, err := s.users.GetUser(ctx, subject)
	if err != nil || u.Password == "" || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPassword)) != nil {
		return constants.ErrInvalidCredentials
	}
	if err := checkPassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.SetUserPassword(ctx, subject, hash); err != nil {
		return err
	}
	return s.users.ClearUserToken(ctx, subject)
}

// HashPassword returns a bcrypt hash. One-way: never store a local password any
// other way.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// TokenHash is the SHA-256 of a raw token, the only form stored.
func TokenHash(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// normalizeEmail is the single spelling of an email: trimmed, lowercased. The
// env root, the login identifier and the OIDC claim all pass through here, so a
// case difference cannot split one identity into two.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func newToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw = hex.EncodeToString(b)
	return raw, TokenHash(raw), nil
}

func checkPassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
