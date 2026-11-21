package commands

import (
	"errors"
	"fmt"
	"noxis/internal/client"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewAuth is the session: where a token is stored, what the server says about it, and how to get rid
// of it again.
func NewAuth(s *Session) *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Sign in to a noxisd, read the session, or forget it",
	}
	auth.AddCommand(newSignin(s), newStatus(s), newSignout(s), newUse(), newLs())
	return auth
}

func newSignin(s *Session) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "signin <token>",
		Short: "Store a token for a server, and prove the server accepts it",
		Long: "Stores the token in " + configPathHint() + " at 0600 and asks the server who it is. " +
			"A token the server refuses is never stored.",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			token := strings.TrimSpace(args[0])
			if token == "" {
				return errors.New("signin needs a token: noxis auth signin <token> --api-url <url>")
			}
			// --profile is the one switch every command shares, so it names the profile here too: signing
			// in to a second server should not mean learning a second flag.
			name := first(name, s.Profile, "default")
			apiURL := first(s.APIURL, os.Getenv("API_URL"), constants.DefaultAPIURL)
			api, err := client.New(apiURL, token)
			if err != nil {
				return err
			}
			identity, err := api.Whoami(cmd.Context())
			if err != nil {
				// The token is not printed, not even when the server refuses it.
				return fmt.Errorf("the server at %s did not accept that token: %w", apiURL, err)
			}

			cfg, err := dotconfig.Load()
			if err != nil {
				return err
			}
			cfg.Put(name, dotconfig.Profile{
				APIURL:    apiURL,
				Token:     token,
				Caller:    identity.Name,
				Scopes:    identity.Scopes,
				CheckedAt: time.Now().UTC(),
			})
			if err := cfg.Save(); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "signed in to %s as %s (%s) as profile %q, stored in %s\n",
				apiURL, identity.Name, strings.Join(identity.Scopes, ","), name, cfg.FilePath())
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "the profile to store the token under (the same as --profile; --profile wins)")
	return cmd
}

func newStatus(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print the server, the caller and the scopes of the stored token",
		Args:  cobra.NoArgs,
		Long: "Prints where the session points and what it is allowed to do, and never the token. " +
			"The server is asked again, so a revoked token shows up here first.",
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := dotconfig.Load()
			if err != nil {
				return err
			}
			name := first(s.Profile, cfg.Current)
			if name == "" {
				return fmt.Errorf("not signed in: run `noxis auth signin <token> --api-url %s`", constants.DefaultAPIURL)
			}
			// Which profile this is, and whether it is the one a bare command would use: a machine can
			// hold several sessions, and "which one am I talking to" has to be answerable.
			mark := ""
			if name == cfg.Current {
				mark = " (current)"
			}
			stored, err := cfg.Profile(name)
			if err != nil {
				return fmt.Errorf("%w: %q is not in %s", dotconfig.ErrNoProfile, name, cfg.FilePath())
			}

			api, err := client.New(stored.APIURL, stored.Token)
			if err != nil {
				return err
			}
			identity, err := api.Whoami(cmd.Context())
			if err != nil {
				return fmt.Errorf("the server at %s did not accept the stored token: run `noxis auth signin <token> --api-url %s` (%w)",
					stored.APIURL, stored.APIURL, err)
			}

			stored.Caller, stored.Scopes, stored.CheckedAt = identity.Name, identity.Scopes, time.Now().UTC()
			cfg.Put(name, stored)
			if err := cfg.Save(); err != nil {
				return err
			}

			if s.JSON {
				return PrintJSON(cmd, struct {
					Profile   string    `json:"profile"`
					APIURL    string    `json:"api_url"`
					Caller    string    `json:"caller"`
					Scopes    []string  `json:"scopes"`
					CheckedAt time.Time `json:"checked_at"`
					File      string    `json:"file"`
				}{name, stored.APIURL, identity.Name, identity.Scopes, stored.CheckedAt, cfg.FilePath()})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"profile:\t%s%s\nserver:\t%s\ncaller:\t%s\nscopes:\t%s\nchecked:\tjust now\nfile:\t%s\n",
				name, mark, stored.APIURL, identity.Name, strings.Join(identity.Scopes, ","), cfg.FilePath())
			return err
		},
	}
}

func newSignout(s *Session) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:         "signout",
		Short:       "Remove the stored token",
		Args:        cobra.NoArgs,
		Long:        "Deletes the profile from the config file, token included. There is nothing left to read afterwards.",
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := dotconfig.Load()
			if err != nil {
				return err
			}
			target := first(name, s.Profile, cfg.Current)
			if target == "" {
				return errors.New("not signed in: there is no stored token to remove")
			}
			if err := cfg.Forget(target); err != nil {
				return fmt.Errorf("%w: %q is not in %s", dotconfig.ErrNoProfile, target, cfg.FilePath())
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "signed out of %s (%s)\n", target, cfg.FilePath())
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "which profile to remove (default the current one)")
	return cmd
}

// newUse switches which stored profile the next command is about, without signing in again: both tokens
// are already on disk, and a person moving between a staging and a production server should not have to
// paste one of them back in.
func newUse() *cobra.Command {
	return &cobra.Command{
		Use:         "use <profile>",
		Short:       "Make a stored profile the one every command uses",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := dotconfig.Load()
			if err != nil {
				return err
			}
			if err := cfg.Use(args[0]); err != nil {
				return fmt.Errorf("%w — stored profiles: %s", err, strings.Join(cfg.Names(), ", "))
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			stored, err := cfg.Profile(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "profile:\t%s\nserver:\t%s\ncaller:\t%s\n",
				args[0], stored.APIURL, stored.Caller)
			return err
		},
	}
}

// newLs answers "which sessions does this machine hold, and which one is in use": the name a command takes
// as --profile, where it points, and who it is signed in as.
func newLs() *cobra.Command {
	return &cobra.Command{
		Use:         "ls",
		Aliases:     []string{"list", "profiles"},
		Short:       "List the stored profiles, marking the one in use",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := dotconfig.Load()
			if err != nil {
				return err
			}
			if len(cfg.Profiles) == 0 {
				return fmt.Errorf("not signed in: run `noxis auth signin <token> --api-url %s`", constants.DefaultAPIURL)
			}
			for _, name := range cfg.Names() {
				stored := cfg.Profiles[name]
				mark := " "
				if name == cfg.Current {
					mark = "*"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %-10s %-34s %s (%s)\n",
					mark, name, stored.APIURL, stored.Caller, strings.Join(stored.Scopes, ",")); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// configPathHint names the file without failing when the home directory cannot be found: a message
// that cannot be built must not become the error a human reads.
func configPathHint() string {
	path, err := dotconfig.Path()
	if err != nil {
		return "~/.config/noxis/config.json"
	}
	return path
}
