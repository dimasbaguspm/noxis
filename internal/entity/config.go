package entity

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Env var names. Env is read exactly once, here; nothing else in the
// codebase talks to the environment.
const (
	EnvMaster        = "NOXIS_MASTER"         // master address: slave/assign/status/sync
	EnvNode          = "NOXIS_NODE"           // this node's id: slave
	EnvListen        = "NOXIS_LISTEN"         // master listen addr: master
	EnvDB            = "NOXIS_DB"             // sqlite path: master
	EnvOTELCollector = "NOXIS_OTEL_COLLECTOR" // OTLP endpoint (optional)
	EnvOTELName      = "NOXIS_OTEL_NAME"      // service name reported to OTEL (optional)
)

// Config is the process environment as a struct. No defaults: every field a
// command needs must be present or Require fails listing the gaps.
type Config struct {
	Master        string `validate:"required"`
	Node          string `validate:"required"`
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
		Node:          os.Getenv(EnvNode),
		Listen:        os.Getenv(EnvListen),
		DB:            os.Getenv(EnvDB),
		OTELCollector: os.Getenv(EnvOTELCollector),
		OTELName:      os.Getenv(EnvOTELName),
	}
}

// Require validates the subset of fields a command needs and aggregates all
// missing variables into one error. Validation rules live in the struct tags;
// this is the per-command slice of them.
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
	case EnvNode:
		return c.Node
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
		EnvNode + "=" + c.Node,
		EnvListen + "=" + c.Listen,
		EnvDB + "=" + c.DB,
		EnvOTELCollector + "=" + c.OTELCollector,
		EnvOTELName + "=" + c.OTELName,
	}, " ")
}
