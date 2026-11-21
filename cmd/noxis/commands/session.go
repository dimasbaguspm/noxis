// Package commands is the client face of the binary: one file per command group, and the session
// they all share. A command reads the flags the root parsed, calls one endpoint through the typed
// client, and prints what came back. It never opens the state file and never reaches a database —
// the API is the only writer, and this package is one of its callers.
package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"noxis/internal/client"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Session is what every command shares: where it points, which token it holds, and what the boot
// check resolved. The root command fills it from the persistent flags; nothing else writes it.
type Session struct {
	// From the flags.
	APIURL  string
	Token   string
	Profile string
	JSON    bool

	// Resolved by Prepare.
	Cfg        dotconfig.Config
	StoredName string
	FromStore  bool
	Name       string
	Scopes     []string
}

// Prepare decides which server and which token this command talks to, and proves the token is still
// one the server accepts. Precedence: the flag, then the stored profile, then the environment — the
// first is what a human typed now, the last is what CI sets and nothing else should.
func (s *Session) Prepare(cmd *cobra.Command) error {
	cfg, err := dotconfig.Load()
	if err != nil {
		return err
	}
	s.Cfg = cfg

	name := s.Profile
	if name == "" {
		name = cfg.Current
	}
	var stored dotconfig.Profile
	if name != "" {
		profile, err := cfg.Profile(name)
		if err != nil && s.Profile != "" {
			// An explicitly named profile that is not there is a mistake, not a reason to fall back.
			return fmt.Errorf("%w: %q is not in %s", dotconfig.ErrNoProfile, name, cfg.FilePath())
		}
		if err == nil {
			stored, s.StoredName = profile, name
		}
	}

	flagURL, flagToken := s.APIURL, s.Token
	envToken := os.Getenv("API_TOKEN")
	s.APIURL = first(flagURL, stored.APIURL, os.Getenv("API_URL"), constants.DefaultAPIURL)
	s.Token = first(flagToken, stored.Token, envToken)
	s.FromStore = flagToken == "" && envToken == "" && stored.Token != ""

	if s.Token == "" {
		return fmt.Errorf("not signed in to %s: run `noxis auth signin <token> --api-url %s`",
			s.APIURL, s.APIURL)
	}
	if !s.FromStore {
		// A token in a shell variable is in the shell's history and in every child process. Say so
		// once, without ever printing the token itself.
		s.note(cmd, "using a token from --token or $API_TOKEN; `noxis auth signin` keeps it out of both")
		return nil
	}
	if cmd.Annotations[constants.AnnoNoVerify] == "true" {
		return nil
	}
	return s.verify(cmd)
}

// verify asks the server who the stored token is, unless it asked recently. A token the server
// refuses is reported as what it is — sign in again — instead of as a 401 in the middle of a
// rehearsal nobody can explain.
func (s *Session) verify(cmd *cobra.Command) error {
	stored, err := s.Cfg.Profile(s.StoredName)
	if err != nil {
		// No stored profile is not a failed verification: there is nothing to verify, and the command
		// that needed a session says so itself.
		return nil //nolint:nilerr // see above
	}
	if stored.Caller != "" && time.Since(stored.CheckedAt) < constants.AuthCheckWindow {
		s.Name, s.Scopes = stored.Caller, stored.Scopes
		return nil
	}

	api, err := s.Client()
	if err != nil {
		return err
	}
	identity, err := api.Whoami(cmd.Context())
	if err != nil {
		var refused *client.Error
		if errors.As(err, &refused) && refused.Status == http.StatusUnauthorized {
			return fmt.Errorf("the server at %s no longer accepts the token stored for %q: run `noxis auth signin <token> --api-url %s`",
				s.APIURL, s.StoredName, s.APIURL)
		}
		return err
	}
	stored.Caller, stored.Scopes, stored.CheckedAt = identity.Name, identity.Scopes, time.Now().UTC()
	s.Cfg.Put(s.StoredName, stored)
	s.Name, s.Scopes = identity.Name, identity.Scopes
	// The cache is a courtesy: a config file that cannot be written must not fail the command the
	// human actually asked for.
	if err := s.Cfg.Save(); err != nil {
		s.note(cmd, "the identity could not be cached: %v", err)
	}
	return nil
}

// Client builds the client a command works through.
func (s *Session) Client() (*client.Client, error) { return client.New(s.APIURL, s.Token) }

// note writes a courtesy to stderr: the advice a person gets and a machine must not have to skip.
// With --json it is a JSON line, so a reader of stderr gets documents end to end.
func (s *Session) note(cmd *cobra.Command, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if !s.JSON {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "noxis: "+message)
		return
	}
	if buf, err := json.Marshal(map[string]string{"warning": message}); err == nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), string(buf))
	}
}

// PrintJSON is the machine shape: the same fields the API answered with.
func PrintJSON(cmd *cobra.Command, v any) error {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(buf))
	return err
}

// first is the precedence rule in one line: the first value that is set wins.
func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
