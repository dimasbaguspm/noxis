// Package slave is the worker daemon. `register slave` captures the node
// identity once; `run slave` reads it and needs no env at all.
package slave

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Identity struct {
	ID   string `json:"id"`
	GRPC string `json:"grpc"` // master's internal transport addr
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
	if id.ID == "" || id.GRPC == "" {
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
