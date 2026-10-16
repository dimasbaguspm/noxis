// Package config reads and validates the process environment.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Env is read exactly once, here; nothing else talks to the environment.
const (
	EnvListen          = "NOXIS_LISTEN"            // REST listen addr (default :9000)
	EnvDB              = "NOXIS_DB"                // sqlite path (default /data/noxis.db)
	EnvSecretKey       = "NOXIS_SECRET_KEY"        // AES key for passwords at rest (64 hex chars; required)
	EnvSchedulerKey    = "NOXIS_SCHEDULER_API_KEY" // external scheduler: shared api key it presents as a bearer token
	EnvSchedulerSecret = "NOXIS_SCHEDULER_SECRET"  // external scheduler: HMAC secret signing the trigger request
	EnvOTELCollector   = "NOXIS_OTEL_COLLECTOR"    // OTLP endpoint (optional)
	EnvOTELName        = "NOXIS_OTEL_NAME"         // service name reported to OTEL (optional)

	// OIDC: when the issuer is set, the API accepts OIDC tokens; it is additive
	// to local accounts. Empty leaves OIDC off.
	EnvOIDCIssuer      = "NOXIS_OIDC_ISSUER"
	EnvOIDCClientID    = "NOXIS_OIDC_CLIENT_ID"
	EnvOIDCAudience    = "NOXIS_OIDC_AUDIENCE"     // defaults to the client id
	EnvOIDCGroupsClaim = "NOXIS_OIDC_GROUPS_CLAIM" // defaults to "groups"
	EnvOIDCAdminGroup  = "NOXIS_OIDC_ADMIN_GROUP"  // implicit global admin
)

// Defaults keep a first instance runnable with no env at all.
const (
	defaultListen = ":9000"
	defaultDB     = "/data/noxis.db"
)

// Config is the resolved process environment: defaults applied, nothing else
// reads os.Getenv.
type Config struct {
	Listen          string
	DB              string
	SecretKey       string
	SchedulerKey    string
	SchedulerSecret string
	OTELCollector   string
	OTELName        string

	OIDCIssuer      string
	OIDCClientID    string
	OIDCAudience    string
	OIDCGroupsClaim string
	OIDCAdminGroup  string
}

// LoadConfig reads every env var exactly once, per process, folding in defaults.
func LoadConfig() Config {
	cfg := Config{
		Listen:          firstNonEmpty(os.Getenv(EnvListen), defaultListen),
		DB:              firstNonEmpty(os.Getenv(EnvDB), defaultDB),
		SecretKey:       os.Getenv(EnvSecretKey),
		SchedulerKey:    os.Getenv(EnvSchedulerKey),
		SchedulerSecret: os.Getenv(EnvSchedulerSecret),
		OTELCollector:   os.Getenv(EnvOTELCollector),
		OTELName:        os.Getenv(EnvOTELName),

		OIDCIssuer:      os.Getenv(EnvOIDCIssuer),
		OIDCClientID:    os.Getenv(EnvOIDCClientID),
		OIDCAudience:    os.Getenv(EnvOIDCAudience),
		OIDCGroupsClaim: os.Getenv(EnvOIDCGroupsClaim),
		OIDCAdminGroup:  os.Getenv(EnvOIDCAdminGroup),
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

func (c Config) String() string {
	return strings.Join([]string{
		EnvListen + "=" + c.Listen,
		EnvDB + "=" + c.DB,
		EnvSecretKey + "=" + redact(c.SecretKey),
		EnvSchedulerKey + "=" + redact(c.SchedulerKey),
		EnvSchedulerSecret + "=" + redact(c.SchedulerSecret),
		EnvOTELCollector + "=" + c.OTELCollector,
		EnvOTELName + "=" + c.OTELName,
		EnvOIDCIssuer + "=" + c.OIDCIssuer,
		EnvOIDCClientID + "=" + c.OIDCClientID,
		EnvOIDCAudience + "=" + c.OIDCAudience,
		EnvOIDCGroupsClaim + "=" + c.OIDCGroupsClaim,
		EnvOIDCAdminGroup + "=" + c.OIDCAdminGroup,
	}, " ")
}

// LogState logs the deployment-relevant facts once, at boot: how secrets and
// auth are configured. Field-level warnings live here, not in command code, so
// a config decision is decided in exactly one place.
func LogState(ctx context.Context, log *slog.Logger, c Config) {
	log.InfoContext(ctx, "connection passwords are encrypted at rest")
	if c.OIDCIssuer == "" {
		log.InfoContext(ctx, "OIDC authentication disabled")
	} else {
		log.InfoContext(ctx, "OIDC authentication enabled", "issuer", c.OIDCIssuer, "adminGroup", c.OIDCAdminGroup)
	}
	if c.SchedulerKey == "" || c.SchedulerSecret == "" {
		log.InfoContext(ctx, "scheduler trigger disabled; set NOXIS_SCHEDULER_API_KEY and NOXIS_SCHEDULER_SECRET to drive scheduled changes")
	} else {
		log.InfoContext(ctx, "scheduler trigger enabled")
	}
}

// redact shows presence, never the value: a config dump must not leak a secret.
func redact(value string) string {
	if value == "" {
		return ""
	}
	return "<redacted>"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// AuthInfo is the public OIDC configuration the console needs to start login.
// The issuer and client id are not secrets.
type AuthInfo struct {
	Enabled  bool   `json:"enabled"`
	Issuer   string `json:"issuer,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}
