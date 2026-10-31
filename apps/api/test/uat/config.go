package uat

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// target maps a test driver to its compose service and resolved credentials.
type target struct {
	Driver   string
	Host     string // the compose service name
	Port     int
	User     string
	Password string
	Database string
}

// loadTargets reads the resolved compose config once and fills the package vars the suite drives the stack with.
func loadTargets() error {
	out, err := exec.Command("docker", "compose", "-f", stackFile, "--profile", "all", "config", "--format", "json").Output()
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

	// Match each driver to the service whose image stands in for it, so the service name and its port come from compose, not from a literal here.
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
		"postgres": "postgres", "mysql": "mysql", "mariadb": "mariadb", "mongo": "mongo",
		"cockroach": "cockroachdb/cockroach",
		"redis":     "redis", "valkey": "valkey",
		"dragonfly":     "docker.dragonflydb.io/dragonflydb/dragonfly",
		"couchdb":       "couchdb",
		"elasticsearch": "docker.elastic.co/elasticsearch/elasticsearch",
		"meilisearch":   "getmeili/meilisearch",
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
	adminEmail = env("noxis", "NOXIS_ADMIN_EMAIL")
	adminPassword = env("noxis", "NOXIS_ADMIN_PASSWORD")
	schedulerKey = env("noxis", "NOXIS_SCHEDULER_API_KEY")
	schedulerSecret = env("noxis", "NOXIS_SCHEDULER_SECRET")

	targets = map[string]target{
		"postgres": {
			Driver: "postgres", Host: host["postgres"], Port: port["postgres"],
			User:     env(host["postgres"], "POSTGRES_USER"),
			Password: env(host["postgres"], "POSTGRES_PASSWORD"), Database: env(host["postgres"], "POSTGRES_DB"),
		},
		"mysql": {
			Driver: "mysql", Host: host["mysql"], Port: port["mysql"],
			User: "root", Password: env(host["mysql"], "MYSQL_ROOT_PASSWORD"),
			Database: env(host["mysql"], "MYSQL_DATABASE"),
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
		"dragonfly": {
			Driver: "dragonfly", Host: host["dragonfly"], Port: port["dragonfly"], Database: "0",
		},
		"couchdb": {
			Driver: "couchdb", Host: host["couchdb"], Port: port["couchdb"],
			User:     env(host["couchdb"], "COUCHDB_USER"),
			Password: env(host["couchdb"], "COUCHDB_PASSWORD"), Database: "demo",
		},
		"elasticsearch": {
			Driver: "elasticsearch", Host: host["elasticsearch"], Port: port["elasticsearch"], Database: "demo",
		},
		"meilisearch": {
			Driver: "meilisearch", Host: host["meilisearch"], Port: port["meilisearch"],
			Password: env(host["meilisearch"], "MEILI_MASTER_KEY"), Database: "demo",
		},
		"mongo": {
			Driver: "mongo", Host: host["mongo"], Port: port["mongo"],
			User:     env(host["mongo"], "MONGO_INITDB_ROOT_USERNAME"),
			Password: env(host["mongo"], "MONGO_INITDB_ROOT_PASSWORD"), Database: env(host["mongo"], "MONGO_INITDB_DATABASE"),
		},
	}
	// sqlite is file-backed: it has no compose service and no host, so its database is a path on the noxis data volume. The path carries the run, so a volume kept by UAT_KEEP cannot leak state into the next run.
	targets["sqlite"] = target{Driver: "sqlite", Database: fmt.Sprintf("/data/uat-compat-%d.db", time.Now().UnixNano())}
	// cockroach is postgres wire on a single node: root with no password, and defaultdb is the database it always has.
	targets["cockroach"] = target{Driver: "cockroach", Host: host["cockroach"], Port: port["cockroach"], User: "root", Database: "defaultdb"}
	return nil
}
