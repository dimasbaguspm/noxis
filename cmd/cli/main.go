// Command noxis is the client: every command is a call to a running noxisd. It holds no
// database connection, no state file and no rule of its own.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"noxis/internal/client"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// options are the flags every command shares.
type options struct {
	apiURL string
	token  string
	asJSON bool
}

func newRoot() *cobra.Command {
	opts := &options{}
	root := &cobra.Command{
		Use:           "noxis",
		Short:         "Client for the noXis API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&opts.apiURL, "api-url", env("API_URL", "http://localhost:8080"), "the noxisd to call")
	root.PersistentFlags().StringVar(&opts.token, "token", env("API_TOKEN", ""), "bearer token")
	root.PersistentFlags().BoolVar(&opts.asJSON, "json", false, "print the response as json")
	root.AddCommand(newWhoami(opts), newVersion(), newTarget(opts),
		newQuery(opts), newDatabases(opts), newTables(opts), newTable(opts),
		newPropose(opts), newProposals(opts), newProposalShow(opts), newRehearse(opts))
	return root
}

// api builds the client a command works through.
func (o *options) api() (*client.Client, error) { return client.New(o.apiURL, o.token) }

// printJSON is the machine shape: the same fields the API answered with.
func printJSON(cmd *cobra.Command, v any) error {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(buf))
	return err
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
