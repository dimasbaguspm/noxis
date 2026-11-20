// Package middlewares wraps every route: recover a panic, allow the console's origin,
// log the request, and resolve the caller a route's scopes demand.
package middlewares

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
)

type callerKey struct{}

type identity struct {
	caller domain.Caller
	digest [sha256.Size]byte
}

// Authenticator resolves a bearer secret to a declared caller. The comparison is over
// SHA-256 digests, so it is constant time whatever the secret's length, and no secret is
// ever a map key or a log line.
type Authenticator struct {
	identities []identity
	log        *slog.Logger
}

// NewAuthenticator builds the matcher from the env-declared tokens.
func NewAuthenticator(tokens []config.Token, log *slog.Logger) (*Authenticator, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("auth: no token declared")
	}
	a := &Authenticator{identities: make([]identity, 0, len(tokens)), log: log}
	for _, token := range tokens {
		a.identities = append(a.identities, identity{
			caller: domain.Caller{Name: token.Name, Scopes: token.Scopes},
			digest: sha256.Sum256([]byte(token.Secret)),
		})
	}
	return a, nil
}

// Require answers 401 when no declared secret matches and 403 when the caller lacks a
// scope the route names; otherwise the handler receives the caller on its context.
func (a *Authenticator) Require(scopes []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get(constants.HeaderAuthorization)
		caller, ok := a.resolve(header)
		if !ok {
			// "Send a token" and "that token is not mine" are different mistakes and send the reader to
			// different places: the first to the sign-in card, the second to whoever issued the token. A
			// header that is not a bearer is the first mistake, not a rejected one.
			if !isBearer(header) {
				a.log.Warn("refused", "reason", "no token", "path", r.URL.Path)
				respond.Refuse(w, http.StatusUnauthorized, constants.CodeUnauthorized, "a bearer token is required")
				return
			}
			a.log.Warn("refused", "reason", "unknown token", "path", r.URL.Path)
			respond.Refuse(w, http.StatusUnauthorized, constants.CodeUnauthorized,
				"that token is not one this server accepts")
			return
		}
		if !caller.HasAll(scopes) {
			a.log.Warn("refused", "reason", "missing scope", "caller", caller.Name, "needs", scopes, "path", r.URL.Path)
			respond.Refuse(w, http.StatusForbidden, constants.CodeForbidden,
				fmt.Sprintf("%q needs scope %s", caller.Name, strings.Join(scopes, " ")))
			return
		}
		next(w, r.WithContext(WithCaller(r.Context(), caller)))
	}
}

// resolve walks every declared token and keeps the last match, so the time a refusal
// takes does not depend on which token a guess was close to.
func (a *Authenticator) resolve(header string) (domain.Caller, bool) {
	scheme, secret, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, constants.SchemeBearer) {
		return domain.Anonymous, false
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return domain.Anonymous, false
	}
	digest := sha256.Sum256([]byte(secret))
	var (
		match domain.Caller
		ok    bool
	)
	for _, id := range a.identities {
		if subtle.ConstantTimeCompare(digest[:], id.digest[:]) == 1 {
			match, ok = id.caller, true
		}
	}
	return match, ok
}

// isBearer reports whether the header even claims to carry a bearer token. "Send a token" and "I do
// not know that token" are two different 401s, and the difference is this question, not an empty
// string comparison: `Basic abc` and `Bearer ` both belong to the first.
func isBearer(header string) bool {
	scheme, secret, found := strings.Cut(header, " ")
	return found && strings.EqualFold(scheme, constants.SchemeBearer) && strings.TrimSpace(secret) != ""
}

// WithCaller puts the authenticated caller on the context.
func WithCaller(ctx context.Context, caller domain.Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

// Caller reads the caller a route required. A public route sees the anonymous caller.
func Caller(ctx context.Context) domain.Caller {
	if caller, ok := ctx.Value(callerKey{}).(domain.Caller); ok {
		return caller
	}
	return domain.Anonymous
}
