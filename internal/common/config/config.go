// Package config is the single place env is read. No scattered os.Getenv, no
// silent defaults: a command declares what it needs and Require fails loudly
// listing the missing variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	EnvMaster        = "NOXIS_MASTER"         // master address, needed by slave/assign/status/sync
	EnvNode          = "NOXIS_NODE"           // this node's id, needed by slave
	EnvListen        = "NOXIS_LISTEN"         // master listen addr, needed by master
	EnvDB            = "NOXIS_DB"             // sqlite path, needed by master
	EnvOTELCollector = "NOXIS_OTEL_COLLECTOR" // OTLP endpoint e.g. "localhost:4317" (optional)
	EnvOTELName      = "NOXIS_OTEL_NAME"      // service name reported to OTEL (optional)
)

type Config struct {
	Master        string
	Node          string
	Listen        string
	DB            string
	OTELCollector string
	OTELName      string
}

// Load reads every env var once. Missing values are not an error here — each
// command calls Require for the vars it actually needs.
func Load() Config {
	return Config{
		Master:        os.Getenv(EnvMaster),
		Node:          os.Getenv(EnvNode),
		Listen:        os.Getenv(EnvListen),
		DB:            os.Getenv(EnvDB),
		OTELCollector: os.Getenv(EnvOTELCollector),
		OTELName:      os.Getenv(EnvOTELName),
	}
}

// Require fails with one error listing every missing variable a command needs.
func (c Config) Require(envs ...string) error {
	var missing []string
	for _, e := range envs {
		if v := valueOf(c, e); v == "" {
			missing = append(missing, e)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return nil
}

func valueOf(c Config, env string) string {
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
