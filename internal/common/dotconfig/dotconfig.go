// Package dotconfig is the CLI's own file: the profiles a human signed in with, and nothing the
// server ever needs to know. It sits at 0600 because it holds a bearer token, and reading it is the
// only place a token outlives the command that used it.
package dotconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrNoProfile says the file holds no profile the caller asked for. It is a sentinel because the CLI
// answers it with an instruction, not a stack of paths.
var ErrNoProfile = errors.New("no such profile")

// Profile is one server this machine is signed in to.
type Profile struct {
	// APIURL is the noxisd this profile talks to, stored so a command needs no flag.
	APIURL string `json:"api_url"`
	// Token is the bearer token. It is written here and nowhere else.
	Token string `json:"token"`
	// Caller and Scopes are what the server said the token is, last time it was asked.
	Caller string   `json:"caller,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	// CheckedAt is when the server last confirmed the token, so a batch of commands asks once.
	CheckedAt time.Time `json:"checked_at"`
}

// Config is the whole file: the profile in use, and every profile that exists.
type Config struct {
	Current  string             `json:"current_profile,omitempty"`
	Profiles map[string]Profile `json:"profiles"`

	// path is where this config was read from, so Save writes back to the same place.
	path string
}

// Path is where the file lives: $XDG_CONFIG_HOME/noxis/config.json, or the same under ~/.config.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "noxis", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: find the home directory: %w", err)
	}
	return filepath.Join(home, ".config", "noxis", "config.json"), nil
}

// Load reads the file. A file that is not there is an empty config and no error: the first command
// a person runs is the one that creates it.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	empty := Config{Profiles: map[string]Profile{}, path: path}

	//nolint:gosec // the path is this tool's own config file, under the user's config directory
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	config := empty
	if err := json.Unmarshal(raw, &config); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	config.path = path
	if config.Profiles == nil {
		config.Profiles = map[string]Profile{}
	}
	return config, nil
}

// Save writes the file back, creating the directory if needed. It never writes a file readable by
// anyone else, whatever the umask or an older version of this tool left behind.
func (c Config) Save() error {
	path := c.path
	if path == "" {
		found, err := Path()
		if err != nil {
			return err
		}
		path = found
	}
	dir := filepath.Dir(path)
	// 0700: the directory names the token just as loudly as the file does.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	// WriteFile keeps the mode of a file that already exists, so a 0644 file from an earlier build
	// would stay readable. Say what it must be instead.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("config: chmod %s: %w", path, err)
	}
	return nil
}

// Profile reads one profile by name.
func (c Config) Profile(name string) (Profile, error) {
	stored, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("%w: %q", ErrNoProfile, name)
	}
	return stored, nil
}

// Put stores a profile and makes it the current one.
func (c *Config) Put(name string, profile Profile) {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = profile
	c.Current = name
}

// Forget removes a profile. Removing the current one leaves the file with no current profile, which
// is what "signed out" means: the next command says how to sign in.
func (c *Config) Forget(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("%w: %q", ErrNoProfile, name)
	}
	delete(c.Profiles, name)
	if c.Current == name {
		c.Current = ""
	}
	return nil
}

// Path reports where this config came from, or would be written.
func (c Config) FilePath() string { return c.path }
