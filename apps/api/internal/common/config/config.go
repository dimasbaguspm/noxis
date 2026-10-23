// Package config reads and validates the process environment.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// Env is read exactly once, here; nothing else talks to the environment.
const (
	EnvListen          = "NOXIS_LISTEN"            // REST listen addr (default :9000)
	EnvDB              = "NOXIS_DB"                // sqlite path (default /data/noxis.db)
	EnvSecretKey       = "NOXIS_SECRET_KEY"        // AES key for passwords at rest (64 hex chars; required)
	EnvAdminEmail      = "NOXIS_ADMIN_EMAIL"       // root super admin's email: it signs in locally, and blends with the same-email OIDC login
	EnvAdminPassword   = "NOXIS_ADMIN_PASSWORD"    // root super admin password (bcrypt-hashed on create)
	EnvSchedulerKey    = "NOXIS_SCHEDULER_API_KEY" // external scheduler: shared api key it presents as a bearer token
	EnvSchedulerSecret = "NOXIS_SCHEDULER_SECRET"  // external scheduler: HMAC secret signing the trigger request
	EnvOTELCollector   = "NOXIS_OTEL_COLLECTOR"    // OTLP endpoint (optional)
	EnvOTELName        = "NOXIS_OTEL_NAME"         // service name reported to OTEL (optional)
	EnvAccessGrantDays = "NOXIS_ACCESS_GRANT_DAYS" // per-connection grant lifetime (default 120, super never expires)

	// OIDC: when the issuer is set, the API accepts OIDC tokens; it is additive
	// to local accounts. Empty leaves OIDC off.
	EnvOIDCIssuer      = "NOXIS_OIDC_ISSUER"
	EnvOIDCClientID    = "NOXIS_OIDC_CLIENT_ID"
	EnvOIDCAudience    = "NOXIS_OIDC_AUDIENCE"     // defaults to the client id
	EnvOIDCGroupsClaim = "NOXIS_OIDC_GROUPS_CLAIM" // defaults to "groups"
)

// Defaults keep a first instance runnable with no env at all.
const (
	defaultListen          = ":9000"
	defaultDB              = "/data/noxis.db"
	defaultAccessGrantDays = 120
)

// Config is the resolved process environment: defaults applied, nothing else
// reads os.Getenv.
type Config struct {
	Listen          string
	DB              string
	SecretKey       string
	AdminEmail      string
	AdminPassword   string
	SchedulerKey    string
	SchedulerSecret string
	OTELCollector   string
	OTELName        string
	AccessGrantTTL  time.Duration

	OIDCIssuer      string
	OIDCClientID    string
	OIDCAudience    string
	OIDCGroupsClaim string
}

// LoadConfig reads every env var exactly once, per process, folding in defaults.
func LoadConfig() Config {
	cfg := Config{
		Listen:          firstNonEmpty(os.Getenv(EnvListen), defaultListen),
		DB:              firstNonEmpty(os.Getenv(EnvDB), defaultDB),
		SecretKey:       os.Getenv(EnvSecretKey),
		AdminEmail:      os.Getenv(EnvAdminEmail),
		AdminPassword:   os.Getenv(EnvAdminPassword),
		SchedulerKey:    os.Getenv(EnvSchedulerKey),
		SchedulerSecret: os.Getenv(EnvSchedulerSecret),
		OTELCollector:   os.Getenv(EnvOTELCollector),
		OTELName:        os.Getenv(EnvOTELName),
		AccessGrantTTL:  time.Duration(accessGrantDays(os.Getenv(EnvAccessGrantDays))) * 24 * time.Hour,

		OIDCIssuer:      os.Getenv(EnvOIDCIssuer),
		OIDCClientID:    os.Getenv(EnvOIDCClientID),
		OIDCAudience:    os.Getenv(EnvOIDCAudience),
		OIDCGroupsClaim: os.Getenv(EnvOIDCGroupsClaim),
	}
	return cfg
}

// Validate enforces the rules defaults cannot: connection passwords are always
// encrypted at rest, so the secret key is required.
func (c Config) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("%s is required (64 hex characters, e.g. openssl rand -hex 32)", EnvSecretKey)
	}
	return nil
}

// accessGrantDays parses the grant lifetime in days, falling back to the 120-day
// default when the value is absent, unparseable or not positive.
func accessGrantDays(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return defaultAccessGrantDays
	}
	return n
}

// GrantTTL falls back to the constant when a caller built Config by hand.
func (c Config) GrantTTL() time.Duration {
	if c.AccessGrantTTL <= 0 {
		return constants.AccessGrantTTL
	}
	return c.AccessGrantTTL
}

// LogState logs the deployment-relevant facts once, at boot: how secrets and
// auth are configured. Field-level warnings live here, not in command code, so
// a config decision is decided in exactly one place.
func LogState(ctx context.Context, log *slog.Logger, c Config) {
	log.InfoContext(ctx, "connection passwords are encrypted at rest")
	log.InfoContext(ctx, "access grants expire", "ttl", c.GrantTTL().String())
	if c.OIDCIssuer == "" {
		log.InfoContext(ctx, "OIDC authentication disabled")
	} else {
		log.InfoContext(ctx, "OIDC authentication enabled", "issuer", c.OIDCIssuer)
	}
	if c.SchedulerKey == "" || c.SchedulerSecret == "" {
		log.InfoContext(ctx, "scheduler trigger disabled; set NOXIS_SCHEDULER_API_KEY and NOXIS_SCHEDULER_SECRET to drive scheduled changes")
	} else {
		log.InfoContext(ctx, "scheduler trigger enabled")
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// AuthInfo derives the console's auth bootstrap from the config, so no route or
// server wiring hardcodes what is available. The wire shape lives in entity.
func (c Config) AuthInfo() entity.AuthConfig {
	info := entity.AuthConfig{Local: true}
	if c.OIDCIssuer != "" {
		info.OIDC = &entity.OIDCConfig{Issuer: c.OIDCIssuer, ClientID: c.OIDCClientID}
	}
	return info
}
