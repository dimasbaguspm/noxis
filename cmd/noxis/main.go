// Command noxis is the whole product in one binary: `http serve` is the API that owns the state
// file and serves the console it carries, and every other word is the client that talks to that API
// over HTTP. One image, one entrypoint, one URL, no launcher script.
//
// This file is the bootstrap and nothing else: the process, the root command, the four flags every
// command shares, and the boot check that turns them into a session. The command groups live in
// `commands`, the API in `http`, and this package composes them.
package main

import (
	"context"
	"fmt"
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
	// Every other Go CLI answers `--version`, and a version probe reaches for it before it reads a help
	// page. It prints the same two lines `noxis version` does, and it needs no session to do it.
	root.Version = constants.Build()
	root.SetVersionTemplate(fmt.Sprintf("{{.Name}} {{.Version}}\nreleases %s\n", constants.ReleaseURL))
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
	// The update check runs whenever a client is bootstrapped: after whatever the person actually asked
	// for, one line on stderr, and never a failure of the command that ran.
	root.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		commands.Notice(cmd, session)
	}
	// One `help`, and it is ours: cobra adds its own unless it is told which command to use, and two rows
	// called `help` in the same list is the accident this pair of lines exists to prevent.
	help := newHelp(root)
	root.SetHelpCommand(help)
	// `--help` on the root answers the same question `noxis` does — which build, which sessions, which
	// server — and then the map. A subcommand keeps cobra's own page.
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c != root {
			defaultHelp(c, args)
			return
		}
		if err := welcome(c.OutOrStdout()); err != nil {
			return
		}
		_, _ = fmt.Fprintln(c.OutOrStdout())
		printGroups(c.OutOrStdout(), root)
	})
	root.AddCommand(help, http.NewHTTP(buildServer),
		commands.NewAPI(session), commands.NewVersion(), newSkills(session),
		commands.NewAuth(session), commands.NewTarget(session),
		commands.NewQuery(session), commands.NewTable(session),
		// The two older depths of the same question, kept for the scripts that named them.
		commands.Deprecated(commands.NewDatabases(session), "databases <ref|name>", "table <target>"),
		commands.Deprecated(commands.NewTables(session), "tables <ref|name> <schema>", "table <target> <schema>"),
		commands.NewPropose(session), commands.NewProposal(session),
		// The eight top-level verbs the group replaced: they run, they say where they went, and `help`
		// does not list them.
		commands.Deprecated(commands.NewProposals(session), "proposals", "proposal ls"),
		commands.Deprecated(commands.NewProposalShow(session), "show-proposal <ref>", "proposal show"),
		commands.Deprecated(commands.NewRehearse(session), "rehearse <ref>", "proposal rehearse"),
		commands.Deprecated(commands.NewApprove(session), "approve <ref>", "proposal approve"),
		commands.Deprecated(commands.NewReject(session), "reject <ref>", "proposal reject"),
		commands.Deprecated(commands.NewApplyProposal(session), "apply <ref>", "proposal apply"),
		commands.Deprecated(commands.NewWatch(session), "watch <ref>", "proposal watch"),
		commands.Deprecated(commands.NewEvents(session), "events <ref>", "proposal events"),
		commands.Deprecated(commands.NewWhoami(session), "whoami", "auth whoami"),
		commands.Deprecated(commands.NewDescribe(session), "describe", "api describe"))
	return root
}
