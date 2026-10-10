// Package auth verifies OIDC bearer tokens against the issuer's JWKS. It is
// the only place that knows about tokens; callers get an entity.Principal.
package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"noxis/internal/common/config"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const defaultGroupsClaim = "groups"

// OIDC is a configured issuer. A nil *OIDC means authentication is disabled
// (no issuer set) and the caller is treated as open.
type OIDC struct {
	verifier    *oidc.IDTokenVerifier
	groupsClaim string
	adminGroup  string
}

// Satisfier.
var _ interfaces.Verifier = (*OIDC)(nil)

// New discovers the issuer and builds a verifier. An empty issuer returns
// (nil, nil): the caller decides whether to run open.
func New(ctx context.Context, cfg config.Config) (*OIDC, error) {
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
		adminGroup:  cfg.OIDCAdminGroup,
	}, nil
}

// Verify checks the token's signature, issuer, audience and expiry, then reads
// the subject and groups off its claims.
func (o *OIDC) Verify(ctx context.Context, rawToken string) (entity.Principal, error) {
	tok, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return entity.Principal{}, fmt.Errorf("verify token: %w", err)
	}
	var claims map[string]any
	if err := tok.Claims(&claims); err != nil {
		return entity.Principal{}, fmt.Errorf("read claims: %w", err)
	}
	return principalFromClaims(tok.Subject, claims, o.groupsClaim, o.adminGroup), nil
}

// principalFromClaims builds the caller. The groups claim may be absent (no
// groups) or a JSON array; anything else is ignored rather than trusted.
func principalFromClaims(subject string, claims map[string]any, groupsClaim, adminGroup string) entity.Principal {
	groups := stringSlice(claims[groupsClaim])
	return entity.Principal{
		Subject: subject,
		// display name: "name", falling back to the username claim
		Name:   firstNonEmpty(stringClaim(claims["name"]), stringClaim(claims["preferred_username"])),
		Email:  stringClaim(claims["email"]),
		Groups: groups,
		Admin:  adminGroup != "" && slices.Contains(groups, adminGroup),
	}
}

func stringClaim(value any) string {
	s, _ := value.(string)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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
