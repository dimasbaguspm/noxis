package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/constants"
)

func newWhoami(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Ask the API which caller this token is",
		Args:  cobra.NoArgs,
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
		Short: "Print the client version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "noxis %s\n", constants.Version)
			return err
		},
	}
}
