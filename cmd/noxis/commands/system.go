package commands

import (
	"context"
	"fmt"
	"noxis/internal/constants"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewWhoami is the check a person runs first: one call, one line, no cached answer.
func NewWhoami(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Ask the API which caller this token is",
		Args:  cobra.NoArgs,
		// The command is the check: running the boot check first would ask the same question twice.
		Annotations: map[string]string{constants.AnnoNoVerify: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			identity, err := api.Whoami(cmd.Context())
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, identity)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", identity.Name, strings.Join(identity.Scopes, ","))
			return err
		},
	}
}

// NewVersion is a fact about the binary, so it needs no session and no network. It names the build and the
// page the update notices point at; whether a newer one exists is the server's answer, not this binary's.
func NewVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		Args:  cobra.NoArgs,
		// A version is a fact about the binary, not about a session: it answers before signing in,
		// which is what a container's first health check needs.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "noxis %s\n", constants.Build()); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "releases %s\n", constants.ReleaseURL)
			return err
		},
	}
}

// Notice is the update check: one line on stderr when the deployment says a newer build was published.
// It runs after every command that reached the server, it swallows every error — a check that fails must
// not fail the command that was asked for — and it writes to stderr so `--json` stays parseable.
func Notice(cmd *cobra.Command, s *Session) {
	api, err := s.Client()
	if err != nil {
		return
	}
	// The check is a courtesy: it must never hold a command open while GitHub thinks about it.
	ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
	defer cancel()
	info, err := api.Version(ctx)
	if err != nil || !info.Newer || info.Latest == nil {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
		"a newer build is out: %s+%s (this is %s) — %s\n",
		info.Latest.Version, info.Latest.Commit, constants.Build(), info.ReleaseURL)
}
