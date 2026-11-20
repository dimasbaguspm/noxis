package main

import (
	"fmt"
	"noxis/internal/constants"
	"strings"

	"github.com/spf13/cobra"
)

func newWhoami(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Ask the API which caller this token is",
		Args:  cobra.NoArgs,
		// The command is the check: running the boot check first would ask the same question twice.
		Annotations: map[string]string{annoNoVerify: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			identity, err := api.Whoami(cmd.Context())
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, identity)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", identity.Name, strings.Join(identity.Scopes, ","))
			return err
		},
	}
}

func newVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		Args:  cobra.NoArgs,
		// A version is a fact about the binary, not about a session: it answers before signing in,
		// which is what a container's first health check needs.
		Annotations: map[string]string{annoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "noxis %s\n", constants.Version)
			return err
		},
	}
}
