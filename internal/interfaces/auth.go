package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Verifier turns a raw bearer token into the caller it represents. The OIDC
// service implements it against the issuer's JWKS; tests use a fake.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (entity.Principal, error)
}
