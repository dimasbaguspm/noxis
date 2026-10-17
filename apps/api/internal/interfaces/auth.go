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
