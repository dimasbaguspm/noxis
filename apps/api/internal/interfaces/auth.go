package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Verifier turns a raw bearer token into the caller it represents. The auth
// service implements it against local accounts and, when configured, OIDC.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (entity.Identity, error)
}

// UserRepository persists local accounts and OIDC identity references, and each
// account's single active session token. Passwords are stored as bcrypt hashes;
// the token as a SHA-256 hash, so the raw credential never touches the database.
// The subject is the key: "local:<username>" or the OIDC sub.
type UserRepository interface {
	// CountUsers reports how many accounts exist (root seeding decides on it).
	CountUsers(ctx context.Context) (int, error)
	// GetUser returns an account by subject, or ErrNotFound.
	GetUser(ctx context.Context, subject string) (*entity.User, error)
	// GetUserByUsername returns a local account by login name, or ErrNotFound.
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	// EnsureSubject records an OIDC identity reference the first time it is
	// seen, so grants and the directory can point at it.
	EnsureSubject(ctx context.Context, subject string) error
	// SaveUser inserts or updates an account (password already hashed).
	SaveUser(ctx context.Context, u entity.User) error
	// SetUserPassword replaces the stored hash.
	SetUserPassword(ctx context.Context, subject, passwordHash string) error
	// SetUserToken makes tokenHash the account's single active token.
	SetUserToken(ctx context.Context, subject, tokenHash string, expiresAt int64) error
	// ClearUserToken logs the account out (idempotent).
	ClearUserToken(ctx context.Context, subject string) error
	// UserByToken returns the enabled account owning an unexpired token, or
	// ErrNotFound.
	UserByToken(ctx context.Context, tokenHash string, now int64) (*entity.User, error)
}
