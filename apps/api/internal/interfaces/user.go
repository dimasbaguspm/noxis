package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// UserRepository persists local accounts and OIDC identity references, and each
// account's single active session token. Passwords are stored as bcrypt hashes;
// the token as a SHA-256 hash, so the raw credential never touches the database.
// The subject is the key: "local:<login>" or the OIDC sub.
type UserRepository interface {
	// CountUsers reports how many accounts exist (root seeding decides on it).
	CountUsers(ctx context.Context) (int, error)
	// GetUser returns an account by subject, or ErrNotFound.
	GetUser(ctx context.Context, subject string) (*entity.User, error)
	// GetUserByEmail returns the account blended to a verified email, or
	// ErrNotFound. It is how an OIDC login joins its local account.
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	// ListUsers returns every account, for the permission editor's member picker.
	ListUsers(ctx context.Context) ([]entity.User, error)
	// GetUserByLogin returns a local account by the identifier its owner types
	// to sign in (its email, or its username), or ErrNotFound.
	GetUserByLogin(ctx context.Context, login string) (*entity.User, error)
	// EnsureSubject records an OIDC identity reference the first time it is
	// seen, so grants and the directory can point at it.
	EnsureSubject(ctx context.Context, subject, email string) error
	// SaveUser inserts or updates an account (password already hashed).
	SaveUser(ctx context.Context, u entity.User) error
	// SetUserPassword replaces the stored hash.
	SetUserPassword(ctx context.Context, subject, passwordHash string) error
	// SetUserToken makes tokenHash the account's single active token.
	SetUserToken(ctx context.Context, subject, tokenHash string, expiresAt int64) error
	// SetUserStatus sets active or banned. A ban also drops the session.
	SetUserStatus(ctx context.Context, subject, status string) error
	// SetUserRole sets the global role (super, admin or member).
	SetUserRole(ctx context.Context, subject, role string) error
	// SetLastLogin records the moment of a successful login.
	SetLastLogin(ctx context.Context, subject string, at int64) error
	// SetSoleSuper makes subject the only super admin, demoting any other. The
	// env root is the only source of that role; admins are left alone.
	SetSoleSuper(ctx context.Context, subject string) error
	// ClearUserToken logs the account out (idempotent).
	ClearUserToken(ctx context.Context, subject string) error
	// UserByToken returns the enabled account owning an unexpired token, or
	// ErrNotFound.
	UserByToken(ctx context.Context, tokenHash string, now int64) (*entity.User, error)
}
