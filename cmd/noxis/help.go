package main

import (
	"context"
	"fmt"
	"io"
	"noxis/internal/client"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// serverProbeTimeout is how long a greeting waits for a server that may be down: long enough for a slow
// laptop network, short enough that nobody rebinds the key.
const serverProbeTimeout = 2 * time.Second

// group is one section of the command list: the commands a person reaches for together.
type group struct {
	title    string
	commands []string
}

// groups is the order a newcomer needs, not the alphabetical order cobra prints.
var groups = []group{
	{"session", []string{"auth", "whoami", "version"}},
	{"this binary", []string{"http", "web"}},
	{"targets", []string{"target"}},
	{"reading", []string{"query", "databases", "tables", "table"}},
	{"proposals", []string{"propose", "proposals", "show-proposal", "watch", "rehearse", "approve", "reject", "apply", "events"}},
	{"the API itself", []string{"describe"}},
}

// welcome is what a bare `noxis` says: which build this is, which sessions this machine holds, where the
// current one points, and what that server answers when asked. The sessions come from the config file; the
// server line is a short request that is allowed to fail, because a greeting must not hang on a server that
// is down.
func welcome(out io.Writer) error {
	_, err := fmt.Fprintf(out, `noxis %s — data governance for the databases you run.

%s
%s

One binary, three faces:
  noxis http serve      the API, and the only writer of the state file
  noxis web serve       the console, served from this binary
  noxis <command>       the client for that API

Next steps:
  noxis auth signin <token> --api-url %s      # stores a session under "default"
  noxis target add --name prod --host db --database app --username noxis
  noxis propose <target> --title "add a column" --sql "ALTER TABLE users ADD COLUMN note TEXT"

  noxis help            every command, grouped           noxis ?        the same
  --json on any command for an agent to read
`, constants.Build(), profileLines(), serverLine(), constants.DefaultAPIURL)
	return err
}

// profileLines lists the sessions this machine holds, marking the one a bare command uses. A machine with
// none says so and names the command that fixes it, because that is the next thing to type.
func profileLines() string {
	cfg, err := dotconfig.Load()
	if err != nil || len(cfg.Profiles) == 0 {
		return "profiles: none — `noxis auth signin <token> --api-url <url>` stores one"
	}
	lines := make([]string, 0, len(cfg.Profiles)+1)
	lines = append(lines, "profiles:")
	for _, name := range cfg.Names() {
		stored := cfg.Profiles[name]
		mark := " "
		if name == cfg.Current {
			mark = "*"
		}
		who := stored.Caller
		if who == "" {
			who = "unconfirmed"
		}
		lines = append(lines, fmt.Sprintf("  %s %-9s %-30s %s", mark, name, stored.APIURL, who))
	}
	return strings.Join(lines, "\n")
}

// serverLine asks the current session's server which build it is. It is best effort: the answer is useful,
// and its absence is not an error on a greeting.
func serverLine() string {
	cfg, err := dotconfig.Load()
	if err != nil {
		return ""
	}
	profile, err := cfg.Profile(cfg.Current)
	if err != nil || profile.APIURL == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), serverProbeTimeout)
	defer cancel()
	api, err := client.New(profile.APIURL, profile.Token)
	if err != nil {
		return fmt.Sprintf("server: %s — %v", profile.APIURL, err)
	}
	info, err := api.Version(ctx)
	if err != nil {
		return fmt.Sprintf("server: %s did not answer; the CLI works offline", profile.APIURL)
	}
	newer := ""
	if info.Newer && info.Latest != nil {
		newer = fmt.Sprintf(" — newer published: %s+%s", info.Latest.Version, info.Latest.Commit)
	}
	return fmt.Sprintf("server: %s answers %s%s", profile.APIURL, info.Version, newer)
}

// newHelp is the grouped list, and the reason `noxis ?` exists: an agent types `--help`, a person
// types `?`, and both should land on something that reads like a map.
func newHelp(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:     "help [command]",
		Aliases: []string{"?"},
		Short:   "List every command, grouped by what you are trying to do",
		Long:    "Lists every command grouped by task. `noxis help <command>` prints that command's usage.",
		// Help is what a person reaches for when nothing works yet.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				found, rest, err := root.Find(args)
				if err != nil {
					return err
				}
				found.SetOut(cmd.OutOrStdout())
				found.HelpFunc()(found, rest)
				return nil
			}
			if err := welcome(cmd.OutOrStdout()); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
			printGroups(cmd.OutOrStdout(), root)
			return nil
		},
	}
}

// printGroups walks the static grouping and skips a command this build does not have, so the list
// cannot promise something the binary does not ship.
func printGroups(out io.Writer, root *cobra.Command) {
	known := map[string]*cobra.Command{}
	for _, command := range root.Commands() {
		known[command.Name()] = command
	}

	listed := map[string]bool{"help": true}
	for _, g := range groups {
		var lines []string
		for _, name := range g.commands {
			command, ok := known[name]
			if !ok {
				continue
			}
			listed[name] = true
			lines = append(lines, fmt.Sprintf("  %-14s %s", command.Name(), command.Short))
		}
		if len(lines) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s\n%s\n\n", strings.ToUpper(g.title), strings.Join(lines, "\n"))
	}

	// A command nobody grouped is still a command: print it rather than hide it.
	var rest []string
	for name, command := range known {
		if listed[name] || name == "completion" {
			continue
		}
		rest = append(rest, fmt.Sprintf("  %-14s %s", command.Name(), command.Short))
	}
	if len(rest) > 0 {
		sort.Strings(rest)
		_, _ = fmt.Fprintf(out, "OTHER\n%s\n\n", strings.Join(rest, "\n"))
	}
}
