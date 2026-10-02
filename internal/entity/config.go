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
	EnvMaster        = "NOXIS_MASTER"         // REST base URL of the master: CLI/frontend
	EnvBroker        = "NOXIS_BROKER"         // Redpanda address: master + register
	EnvListen        = "NOXIS_LISTEN"         // REST listen addr: master
	EnvDB            = "NOXIS_DB"             // sqlite path: master
	EnvOTELCollector = "NOXIS_OTEL_COLLECTOR" // OTLP endpoint (optional)
	EnvOTELName      = "NOXIS_OTEL_NAME"      // service name reported to OTEL (optional)
)

// Config is the process environment as a struct; no defaults.
type Config struct {
	Master        string `validate:"required"`
	Broker        string `validate:"required"`
	Listen        string `validate:"required"`
	DB            string `validate:"required"`
	OTELCollector string
	OTELName      string
}

var validate = validator.New()

// Load reads every env var exactly once, per process.
func LoadConfig() Config {
	return Config{
		Master:        os.Getenv(EnvMaster),
		Broker:        os.Getenv(EnvBroker),
		Listen:        os.Getenv(EnvListen),
		DB:            os.Getenv(EnvDB),
		OTELCollector: os.Getenv(EnvOTELCollector),
		OTELName:      os.Getenv(EnvOTELName),
	}
}

// Require aggregates the missing variables a command needs, via struct tags.
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
	case EnvMaster:
		return c.Master
	case EnvBroker:
		return c.Broker
	case EnvListen:
		return c.Listen
	case EnvDB:
		return c.DB
	case EnvOTELCollector:
		return c.OTELCollector
	case EnvOTELName:
		return c.OTELName
	}
	return ""
}

func (c Config) String() string {
	return strings.Join([]string{
		EnvMaster + "=" + c.Master,
		EnvListen + "=" + c.Listen,
		EnvDB + "=" + c.DB,
		EnvOTELCollector + "=" + c.OTELCollector,
		EnvOTELName + "=" + c.OTELName,
	}, " ")
}
