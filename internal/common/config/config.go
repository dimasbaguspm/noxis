// Package config is the only place in the tree that reads the environment. Every value
// the process needs is resolved once at boot, and anything malformed is an error rather
// than a default: a server that silently loses its token list is a server nobody can call.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"noxis/internal/common/secret"
	"noxis/internal/constants"
)

// Token is one caller. Its name is the identity that lands in the journal, so it names
// the caller (an agent, a person, a CI job) rather than the permission.
type Token struct {
	Name   string
	Secret string
	Scopes []string
}

// Config is the resolved environment.
type Config struct {
	LogLevel          string
	LogFormat         string
	QueryTimeout      time.Duration
	ProbeTimeout      time.Duration
	OTELEndpoint      string
	OTELServiceName   string
	SecretKey         []byte
	SecretKeyPrevious []byte
	Tokens            []Token
}

var tokenName = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

// Load reads the environment.
func Load() (Config, error) { return LoadFrom(os.Getenv) }

// LoadFrom is Load with the environment injected, so the parse rules are testable.
func LoadFrom(get func(string) string) (Config, error) {
	cfg := Config{
		LogLevel:  or(get("LOG_LEVEL"), "info"),
		LogFormat: or(get("LOG_FORMAT"), "text"),
	}
	var err error
	if cfg.QueryTimeout, err = duration(or(get("QUERY_TIMEOUT"), "30s"), "QUERY_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.ProbeTimeout, err = duration(or(get("PROBE_TIMEOUT"), "5s"), "PROBE_TIMEOUT"); err != nil {
		return Config{}, err
	}
	cfg.OTELEndpoint = or(get("OTEL_EXPORTER_OTLP_ENDPOINT"), "")
	cfg.OTELServiceName = or(get("OTEL_SERVICE_NAME"), "noxis")
	if raw := strings.TrimSpace(get("SECRET_KEY")); raw != "" {
		if cfg.SecretKey, err = secret.ParseKey(raw); err != nil {
			return Config{}, fmt.Errorf("config: SECRET_KEY: %w", err)
		}
	}
	if raw := strings.TrimSpace(get("SECRET_KEY_PREVIOUS")); raw != "" {
		if cfg.SecretKeyPrevious, err = secret.ParseKey(raw); err != nil {
			return Config{}, fmt.Errorf("config: SECRET_KEY_PREVIOUS: %w", err)
		}
	}
	raw, err := tokensRaw(get)
	if err != nil {
		return Config{}, err
	}
	if cfg.Tokens, err = ParseTokens(raw); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ParseTokens reads the inline token JSON: {"<name>": {"secret": "...", "scopes": [...]}}.
// A map rather than a list, so two callers cannot accidentally share a name.
func ParseTokens(raw string) ([]Token, error) {
	var declared map[string]struct {
		Secret string   `json:"secret"`
		Scopes []string `json:"scopes"`
	}
	if err := json.Unmarshal([]byte(raw), &declared); err != nil {
		return nil, fmt.Errorf("config: TOKENS is not valid JSON: %w", err)
	}
	if len(declared) == 0 {
		return nil, fmt.Errorf("config: TOKENS declares no caller")
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Token, 0, len(names))
	bySavedSecret := make(map[string]string, len(names))
	for _, name := range names {
		decl := declared[name]
		if !tokenName.MatchString(name) {
			return nil, fmt.Errorf("config: token name %q must match %s", name, tokenName)
		}
		if len(decl.Secret) < constants.MinTokenSecretLen {
			return nil, fmt.Errorf("config: token %q needs a secret of at least %d characters", name, constants.MinTokenSecretLen)
		}
		if len(decl.Scopes) == 0 {
			return nil, fmt.Errorf("config: token %q declares no scope", name)
		}
		scopes := make([]string, 0, len(decl.Scopes))
		seen := make(map[string]bool, len(decl.Scopes))
		for _, scope := range decl.Scopes {
			scope = strings.TrimSpace(scope)
			if !constants.IsScope(scope) {
				return nil, fmt.Errorf("config: token %q names unknown scope %q (known: %s)",
					name, scope, strings.Join(constants.Scopes, ", "))
			}
			if !seen[scope] {
				seen[scope] = true
				scopes = append(scopes, scope)
			}
		}
		if other, duplicate := bySavedSecret[decl.Secret]; duplicate {
			return nil, fmt.Errorf("config: tokens %q and %q share a secret, so neither is an identity", other, name)
		}
		bySavedSecret[decl.Secret] = name
		out = append(out, Token{Name: name, Secret: decl.Secret, Scopes: scopes})
	}
	return out, nil
}

// tokensRaw reads the token JSON from TOKENS_FILE when set, else from TOKENS.
func tokensRaw(get func(string) string) (string, error) {
	if path := strings.TrimSpace(get("TOKENS_FILE")); path != "" {
		b, err := os.ReadFile(path) //nolint:gosec // the operator names this path on purpose
		if err != nil {
			return "", fmt.Errorf("config: TOKENS_FILE: %w", err)
		}
		return string(b), nil
	}
	if raw := strings.TrimSpace(get("TOKENS")); raw != "" {
		return raw, nil
	}
	return "", fmt.Errorf(`config: TOKENS is required: declare at least one caller, for example {"agent":{"secret":"...","scopes":["read","propose"]}}`)
}

func or(value, fallback string) string {
	if v := strings.TrimSpace(value); v != "" {
		return v
	}
	return fallback
}

func duration(s, name string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config: %s: %w", name, err)
	}
	return d, nil
}
