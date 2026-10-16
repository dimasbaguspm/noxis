// Package config reads and validates the process environment.
package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Env is read exactly once, here; nothing else talks to the environment.
const (
	EnvListen          = "NOXIS_LISTEN"            // REST listen addr
	EnvDB              = "NOXIS_DB"                // sqlite path (connection registry)
	EnvSecretKey       = "NOXIS_SECRET_KEY"        // AES key for passwords at rest (64 hex chars; empty = plaintext)
	EnvSchedulerKey    = "NOXIS_SCHEDULER_API_KEY" // external scheduler: shared api key it presents as a bearer token
	EnvSchedulerSecret = "NOXIS_SCHEDULER_SECRET"  // external scheduler: HMAC secret signing the trigger request
	EnvOTELCollector   = "NOXIS_OTEL_COLLECTOR"    // OTLP endpoint (optional)
	EnvOTELName        = "NOXIS_OTEL_NAME"         // service name reported to OTEL (optional)

	// OIDC: when the issuer is set, the API authenticates every caller against
	// it; when it is empty, noxis runs open (single-operator default).
	EnvOIDCIssuer      = "NOXIS_OIDC_ISSUER"
	EnvOIDCClientID    = "NOXIS_OIDC_CLIENT_ID"
	EnvOIDCAudience    = "NOXIS_OIDC_AUDIENCE"     // defaults to the client id
	EnvOIDCGroupsClaim = "NOXIS_OIDC_GROUPS_CLAIM" // defaults to "groups"
	EnvOIDCAdminGroup  = "NOXIS_OIDC_ADMIN_GROUP"  // implicit global admin
)

// Config is the process environment; no defaults.
type Config struct {
	Listen          string `validate:"required"`
	DB              string `validate:"required"`
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

var validate = validator.New()

// LoadConfig reads every env var exactly once, per process.
func LoadConfig() Config {
	return Config{
		Listen:          os.Getenv(EnvListen),
		DB:              os.Getenv(EnvDB),
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
}

// Require aggregates the missing variables a command needs.
func (c Config) Require(envs ...string) error {
	var errs []error
	for _, env := range envs {
		if err := validate.Var(fieldOf(c, env), "required"); err != nil {
			errs = append(errs, fmt.Errorf("%s is required", env))
		}
	}
	return errors.Join(errs...)
}

func fieldOf(c Config, env string) string {
	switch env {
	case EnvListen:
		return c.Listen
	case EnvDB:
		return c.DB
	case EnvSecretKey:
		return c.SecretKey
	case EnvSchedulerKey:
		return c.SchedulerKey
	case EnvSchedulerSecret:
		return c.SchedulerSecret
	case EnvOTELCollector:
		return c.OTELCollector
	case EnvOTELName:
		return c.OTELName
	case EnvOIDCIssuer:
		return c.OIDCIssuer
	case EnvOIDCClientID:
		return c.OIDCClientID
	case EnvOIDCAudience:
		return c.OIDCAudience
	case EnvOIDCGroupsClaim:
		return c.OIDCGroupsClaim
	case EnvOIDCAdminGroup:
		return c.OIDCAdminGroup
	}
	return ""
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
	if c.SecretKey == "" {
		log.WarnContext(ctx, "NOXIS_SECRET_KEY is not set; connection passwords are stored in plaintext")
	} else {
		log.InfoContext(ctx, "connection passwords are encrypted at rest")
	}
	if c.OIDCIssuer == "" {
		log.WarnContext(ctx, "NOXIS_OIDC_ISSUER is not set; the API is unauthenticated")
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

// AuthInfo is the public OIDC configuration the console needs to start login.
// The issuer and client id are not secrets.
type AuthInfo struct {
	Enabled  bool   `json:"enabled"`
	Issuer   string `json:"issuer,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}
