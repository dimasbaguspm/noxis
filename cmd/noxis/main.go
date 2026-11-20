// Command noxis is the whole product in one binary: `http serve` is the API that owns the state
// file, `web serve` is the console, and every other word is the client that talks to the API over
// HTTP. One image, one entrypoint, no launcher script.
//
// This file is the bootstrap and nothing else: the process, the root command, the four flags every
// command shares, and the boot check that turns them into a session. The command groups live in
// `commands`, the API in `http`, and this package composes them.
package main

import (
	"context"
	"noxis/cmd/noxis/commands"
	"noxis/cmd/noxis/http"
	"noxis/internal/constants"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	// os.Exit skips every deferred call, so the signal context is released by run(), not here.
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root := newRoot()
	if err := root.ExecuteContext(ctx); err != nil {
		// A command that failed still owes an agent a shape: --json is honoured on the way out, not
		// only on data. The exit code stays one because the code travels inside the document.
		commands.Report(os.Stderr, err, jsonOutput(root))
		return 1
	}
	return 0
}

// jsonOutput reads the machine-shape flag back off the tree: run() holds the command, not the session
// the flags were bound to, and a failure has to be shaped the same way the answer would have been.
func jsonOutput(root *cobra.Command) bool {
	flag := root.PersistentFlags().Lookup("json")
	return flag != nil && flag.Value.String() == "true"
}

// newRoot builds the command tree once: one session, filled by the persistent flags, handed to
// every command group.
func newRoot() *cobra.Command {
	session := &commands.Session{}
	root := &cobra.Command{
		Use:           "noxis",
		Short:         "Data governance for the databases you run",
		SilenceUsage:  true,
		SilenceErrors: true,
		// A bare `noxis` is the first thing a person types: it answers with the map, not the usage,
		// and it cannot require the session it is about to tell you how to create.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return welcome(cmd.OutOrStdout())
		},
	}
	root.PersistentFlags().StringVar(&session.APIURL, "api-url", "",
		"the noxisd to call (default $API_URL, then the signed-in profile)")
	root.PersistentFlags().StringVar(&session.Token, "token", "",
		"bearer token, for CI (default $API_TOKEN, then the signed-in profile)")
	root.PersistentFlags().StringVar(&session.Profile, "profile", "",
		"which stored profile to use (default the current one)")
	root.PersistentFlags().BoolVar(&session.JSON, "json", false, "print the response as json")

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if cmd.Annotations[constants.AnnoNoToken] == "true" {
			return nil
		}
		return session.Prepare(cmd)
	}
	root.AddCommand(newHelp(root), http.NewHTTP(buildServer), newWeb(),
		commands.NewWhoami(session), commands.NewVersion(), commands.NewAuth(session),
		commands.NewTarget(session),
		commands.NewQuery(session), commands.NewDatabases(session),
		commands.NewTables(session), commands.NewTable(session),
		commands.NewPropose(session), commands.NewProposals(session),
		commands.NewProposalShow(session), commands.NewRehearse(session),
		commands.NewApprove(session), commands.NewReject(session),
		commands.NewApplyProposal(session), commands.NewEvents(session))
	return root
}
