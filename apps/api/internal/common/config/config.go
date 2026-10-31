// Package config reads and validates the process environment.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/viper"

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

	// OIDC: when the issuer is set, the API accepts OIDC tokens; it is additive to local accounts.
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

// Config is the resolved process environment: defaults applied, nothing else reads os.Getenv.
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

// LoadConfig reads every env var exactly once, per process, folding in defaults. Viper is the reader: a key is named
// once, its default is declared beside it, and an empty variable reads as the default rather than as an empty value.
func LoadConfig() Config {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault(EnvListen, defaultListen)
	v.SetDefault(EnvDB, defaultDB)
	v.SetDefault(EnvAccessGrantDays, defaultAccessGrantDays)
	days := v.GetInt(EnvAccessGrantDays)
	if days <= 0 {
		days = defaultAccessGrantDays
	}
	return Config{
		Listen:          v.GetString(EnvListen),
		DB:              v.GetString(EnvDB),
		SecretKey:       v.GetString(EnvSecretKey),
		AdminEmail:      v.GetString(EnvAdminEmail),
		AdminPassword:   v.GetString(EnvAdminPassword),
		SchedulerKey:    v.GetString(EnvSchedulerKey),
		SchedulerSecret: v.GetString(EnvSchedulerSecret),
		OTELCollector:   v.GetString(EnvOTELCollector),
		OTELName:        v.GetString(EnvOTELName),
		AccessGrantTTL:  time.Duration(days) * 24 * time.Hour,

		OIDCIssuer:      v.GetString(EnvOIDCIssuer),
		OIDCClientID:    v.GetString(EnvOIDCClientID),
		OIDCAudience:    v.GetString(EnvOIDCAudience),
		OIDCGroupsClaim: v.GetString(EnvOIDCGroupsClaim),
	}
}

// Validate enforces the rules defaults cannot: connection passwords are always encrypted at rest, so the secret key is required.
func (c Config) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("%s is required (64 hex characters, e.g. openssl rand -hex 32)", EnvSecretKey)
	}
	return nil
}

// GrantTTL falls back to the constant when a caller built Config by hand.
func (c Config) GrantTTL() time.Duration {
	if c.AccessGrantTTL <= 0 {
		return constants.AccessGrantTTL
	}
	return c.AccessGrantTTL
}

// LogState reports secret and authentication configuration once at boot.
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

// AuthInfo derives the console's auth bootstrap from the config, so no route or server wiring hardcodes what is available.
func (c Config) AuthInfo() entity.AuthConfig {
	info := entity.AuthConfig{Local: true}
	if c.OIDCIssuer != "" {
		info.OIDC = &entity.OIDCConfig{Issuer: c.OIDCIssuer, ClientID: c.OIDCClientID}
	}
	return info
}
