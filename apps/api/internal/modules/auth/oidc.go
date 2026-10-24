// Package auth verifies OIDC bearer tokens against the issuer's JWKS. It is the only place that knows about tokens; callers get an entity.Identity.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"noxis/internal/common/config"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const defaultGroupsClaim = "groups"

// OIDC is a configured issuer.
type OIDC struct {
	verifier    *oidc.IDTokenVerifier
	groupsClaim string
}

// Satisfier.
var _ interfaces.Verifier = (*OIDC)(nil)

// New discovers a configured issuer; an empty issuer disables OIDC.
func New(ctx context.Context, cfg config.Config) (configured *OIDC, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	issuer := strings.TrimSpace(cfg.OIDCIssuer)
	if issuer == "" {
		return nil, nil
	}
	if cfg.OIDCClientID == "" {
		return nil, errors.New("NOXIS_OIDC_CLIENT_ID is required when NOXIS_OIDC_ISSUER is set")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery %s: %w", issuer, err)
	}
	audience := cfg.OIDCAudience
	if audience == "" {
		audience = cfg.OIDCClientID
	}
	groupsClaim := cfg.OIDCGroupsClaim
	if groupsClaim == "" {
		groupsClaim = defaultGroupsClaim
	}
	return &OIDC{
		verifier:    provider.Verifier(&oidc.Config{ClientID: audience}),
		groupsClaim: groupsClaim,
	}, nil
}

// Verify checks the token's signature, issuer, audience, and expiry before reading its claims.
func (o *OIDC) Verify(ctx context.Context, rawToken string) (identity entity.Identity, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	tok, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return entity.Identity{}, fmt.Errorf("verify token: %w", err)
	}
	var claims map[string]any
	if err := tok.Claims(&claims); err != nil {
		return entity.Identity{}, fmt.Errorf("read claims: %w", err)
	}
	return principalFromClaims(tok.Subject, claims, o.groupsClaim), nil
}

// principalFromClaims accepts string groups; only a verified matching email blends an OIDC identity into a local account.
func principalFromClaims(subject string, claims map[string]any, groupsClaim string) entity.Identity {
	groups := stringSlice(claims[groupsClaim])
	return entity.Identity{
		Subject: subject,
		// display name: "name", falling back to the username claim
		Name:   firstNonEmpty(stringClaim(claims["name"]), stringClaim(claims["preferred_username"])),
		Email:  normalizeEmail(stringClaim(claims["email"])),
		Groups: groups,
		Role:   entity.RoleMember,
		// an issuer never grants a global role: OIDC identities are members until the super admin promotes them
		EmailVerified: boolClaim(claims["email_verified"]),
	}
}

func stringClaim(value any) string {
	s, _ := value.(string)
	return s
}

func boolClaim(value any) bool {
	b, _ := value.(bool)
	return b
}

func stringSlice(value any) []string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
