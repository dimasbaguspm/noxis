// Package identity reads and writes the slave's identity file. `register`
// bakes it once; `run slave` needs zero env afterwards (no config is that
// robust).
package identity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Identity struct {
	ID          string `json:"id"`
	Broker      string `json:"broker"`       // Kafka broker address(es)
	TopicPrefix string `json:"topic_prefix"` // Kafka topic prefix (NOXIS_TOPIC_PREFIX)
	Master      string `json:"master"`       // REST base URL of the master (spec refetch on start)
}

// Path is the identity file location (per-user config dir).
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "noxis", "identity.json")
}

func Load() (*Identity, error) {
	raw, err := os.ReadFile(Path())
	if err != nil {
		return nil, fmt.Errorf("read identity: %w (run `noxis register slave <name>` first)", err)
	}
	var id Identity
	if err := json.Unmarshal(raw, &id); err != nil {
		return nil, fmt.Errorf("parse identity: %w", err)
	}
	if id.ID == "" || id.Broker == "" || id.TopicPrefix == "" || id.Master == "" {
		return nil, errors.New("identity is incomplete")
	}
	return &id, nil
}

func Save(id Identity) error {
	dir := filepath.Dir(Path())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	raw, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), raw, 0o600)
}
