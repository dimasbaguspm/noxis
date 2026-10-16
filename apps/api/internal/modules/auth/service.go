package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ErrInvalidCredentials is the single answer to every failed login: it does not
// reveal whether the account exists, is disabled, or had the wrong password.
var ErrInvalidCredentials = errors.New("invalid credentials")

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
		return entity.Identity{}, ErrInvalidCredentials
	}
	if u, err := s.users.UserByToken(ctx, TokenHash(rawToken), s.now().Unix()); err == nil {
		return u.Identity(), nil
	}
	if s.oidc == nil {
		return entity.Identity{}, ErrInvalidCredentials
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

// EnsureRoot creates the root admin from env on a cold install. Once any account
// exists the database wins, so a changed password survives restarts and a stale
// env value never overwrites it.
func (s *Service) EnsureRoot(ctx context.Context, username, password string) error {
	count, err := s.users.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if strings.TrimSpace(username) == "" || password == "" {
		return errors.New("no local accounts exist; set NOXIS_ADMIN_USERNAME and NOXIS_ADMIN_PASSWORD to create the root admin")
	}
	if err := checkPassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.users.SaveUser(ctx, entity.User{
		Subject:  entity.LocalSubject(username),
		Username: username,
		Password: hash,
		Admin:    true,
	})
}

// Login verifies a password and issues the account's single active token.
func (s *Service) Login(ctx context.Context, username, password string) (string, entity.User, error) {
	u, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		// pay the bcrypt cost anyway: timing must not reveal a known username
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", entity.User{}, ErrInvalidCredentials
	}
	if u.Disabled || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return "", entity.User{}, ErrInvalidCredentials
	}
	token, hash, err := newToken()
	if err != nil {
		return "", entity.User{}, err
	}
	if err := s.users.SetUserToken(ctx, u.Subject, hash, s.now().Add(TokenTTL).Unix()); err != nil {
		return "", entity.User{}, err
	}
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
		return ErrInvalidCredentials
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
