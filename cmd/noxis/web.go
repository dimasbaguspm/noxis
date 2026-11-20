package main

import (
	"github.com/spf13/cobra"
)

// newWeb is the console half of the binary. The assets ship inside it (internal/web), so a machine
// that has noxis has the console; the ticket that builds it fills this in.
func newWeb() *cobra.Command {
	web := &cobra.Command{
		Use:   "web",
		Short: "Run the noXis console",
	}
	web.AddCommand(&cobra.Command{
		Use:         "serve",
		Short:       "Serve the console from this binary",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{annoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write([]byte("web: the console is not built into this binary yet (see #789)\n"))
			return err
		},
	})
	return web
}
