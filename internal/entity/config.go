// Package entity is the shared data model across store, wire and REST.
package entity

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Env is read exactly once, here; nothing else talks to the environment.
const (
	EnvListen        = "NOXIS_LISTEN"         // REST listen addr
	EnvDB            = "NOXIS_DB"             // sqlite path (connection registry)
	EnvTickKey       = "NOXIS_TICK_KEY"       // API key for POST /internal/tick (optional)
	EnvTickURL       = "NOXIS_TICK_URL"       // standalone ticker: the /internal/tick URL
	EnvTickInterval  = "NOXIS_TICK_INTERVAL"  // standalone ticker: how often to tick (e.g. 30s)
	EnvOTELCollector = "NOXIS_OTEL_COLLECTOR" // OTLP endpoint (optional)
	EnvOTELName      = "NOXIS_OTEL_NAME"      // service name reported to OTEL (optional)

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
	Listen        string `validate:"required"`
	DB            string `validate:"required"`
	TickKey       string
	TickURL       string
	TickInterval  string
	OTELCollector string
	OTELName      string

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
		Listen:        os.Getenv(EnvListen),
		DB:            os.Getenv(EnvDB),
		TickKey:       os.Getenv(EnvTickKey),
		TickURL:       os.Getenv(EnvTickURL),
		TickInterval:  os.Getenv(EnvTickInterval),
		OTELCollector: os.Getenv(EnvOTELCollector),
		OTELName:      os.Getenv(EnvOTELName),

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
	case EnvTickKey:
		return c.TickKey
	case EnvTickURL:
		return c.TickURL
	case EnvTickInterval:
		return c.TickInterval
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
		EnvTickKey + "=" + c.TickKey,
		EnvTickURL + "=" + c.TickURL,
		EnvTickInterval + "=" + c.TickInterval,
		EnvOTELCollector + "=" + c.OTELCollector,
		EnvOTELName + "=" + c.OTELName,
		EnvOIDCIssuer + "=" + c.OIDCIssuer,
		EnvOIDCClientID + "=" + c.OIDCClientID,
		EnvOIDCAudience + "=" + c.OIDCAudience,
		EnvOIDCGroupsClaim + "=" + c.OIDCGroupsClaim,
		EnvOIDCAdminGroup + "=" + c.OIDCAdminGroup,
	}, " ")
}
