package uat

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// target is a connection the suite registers: the compose service that stands in
// for it and the credentials to reach it. Every value is read from the compose
// file at startup, so the suite and the stack never drift.
type target struct {
	Driver   string
	Host     string // the compose service name
	Port     int
	User     string
	Password string
	Database string
}

// loadTargets reads the resolved compose config once and fills the package vars
// the suite drives the stack with. `docker compose config` renders the env the
// services actually get and needs no daemon.
func loadTargets() error {
	out, err := exec.Command("docker", "compose", "-f", stackFile, "config", "--format", "json").Output()
	if err != nil {
		return fmt.Errorf("compose config: %w", err)
	}
	var cfg struct {
		Services map[string]struct {
			Image       string            `json:"image"`
			Environment map[string]string `json:"environment"`
			Expose      []string          `json:"expose"`
			Ports       []struct {
				Published string `json:"published"`
			} `json:"ports"`
		} `json:"services"`
	}
	if err := json.Unmarshal(out, &cfg); err != nil {
		return fmt.Errorf("compose config decode: %w", err)
	}
	env := func(service, key string) string { return cfg.Services[service].Environment[key] }

	// Match each driver to the service whose image stands in for it, so the
	// service name and its port come from compose, not from a literal here.
	hostOf := func(image string) (string, error) {
		for name, svc := range cfg.Services {
			if strings.HasPrefix(svc.Image, image) {
				return name, nil
			}
		}
		return "", fmt.Errorf("no compose service with image %q", image)
	}
	host, port := map[string]string{}, map[string]int{}
	for driver, image := range map[string]string{
		"postgres": "postgres", "mariadb": "mariadb", "mongo": "mongo",
		"redis": "redis", "valkey": "valkey",
	} {
		h, err := hostOf(image)
		if err != nil {
			return err
		}
		expose := cfg.Services[h].Expose
		if len(expose) == 0 {
			return fmt.Errorf("service %s exposes no port", h)
		}
		p, err := strconv.Atoi(expose[0])
		if err != nil {
			return fmt.Errorf("service %s expose %q: %w", h, expose[0], err)
		}
		host[driver], port[driver] = h, p
	}

	noxis := cfg.Services["noxis"]
	if len(noxis.Ports) == 0 {
		return fmt.Errorf("noxis service publishes no port")
	}
	baseURL = "http://localhost:" + noxis.Ports[0].Published
	adminUser = env("noxis", "NOXIS_ADMIN_USERNAME")
	adminPassword = env("noxis", "NOXIS_ADMIN_PASSWORD")
	schedulerKey = env("noxis", "NOXIS_SCHEDULER_API_KEY")
	schedulerSecret = env("noxis", "NOXIS_SCHEDULER_SECRET")

	targets = map[string]target{
		"postgres": {
			Driver: "postgres", Host: host["postgres"], Port: port["postgres"],
			User: env(host["postgres"], "POSTGRES_USER"),
			Password: env(host["postgres"], "POSTGRES_PASSWORD"), Database: env(host["postgres"], "POSTGRES_DB"),
		},
		"mariadb": {
			Driver: "mariadb", Host: host["mariadb"], Port: port["mariadb"],
			User: "root", Password: env(host["mariadb"], "MARIADB_ROOT_PASSWORD"),
			Database: env(host["mariadb"], "MARIADB_DATABASE"),
		},
		"redis": {
			Driver: "redis", Host: host["redis"], Port: port["redis"], Database: "0",
		},
		"valkey": {
			Driver: "valkey", Host: host["valkey"], Port: port["valkey"], Database: "0",
		},
		"mongo": {
			Driver: "mongo", Host: host["mongo"], Port: port["mongo"],
			User: env(host["mongo"], "MONGO_INITDB_ROOT_USERNAME"),
			Password: env(host["mongo"], "MONGO_INITDB_ROOT_PASSWORD"), Database: env(host["mongo"], "MONGO_INITDB_DATABASE"),
		},
	}
	return nil
}
