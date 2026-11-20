package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
)

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
	{"proposals", []string{"propose", "proposals", "show-proposal", "rehearse", "approve", "reject", "apply", "events"}},
}

// welcome is what a bare `noxis` says: what this binary is, where it points, and the four commands
// that take a person from nothing to a rehearsed proposal. It reads the config file and never the
// network, so it cannot hang and cannot fail on a server that is down.
func welcome(out io.Writer) error {
	_, err := fmt.Fprintf(out, `noxis %s — reviewed access to MySQL and MariaDB.

One binary, three faces:
  noxis http serve      the API, and the only writer of the state file
  noxis web serve       the console, served from this binary
  noxis <command>       the client for that API

%s

Next steps:
  noxis auth signin <token> --api-url %s
  noxis target add --name prod --host db --database app --username noxis
  noxis propose <target-ref> --title "add a column" --sql "ALTER TABLE users ADD COLUMN note TEXT"

  noxis help            every command, grouped           noxis ?        the same
  --json on any command for an agent to read
`, constants.Version, sessionLine(), constants.DefaultAPIURL)
	return err
}

// sessionLine says where the CLI points and who it thinks it is, and never prints the token. A
// config file that cannot be read is "not signed in", not an error on a greeting.
func sessionLine() string {
	cfg, err := dotconfig.Load()
	if err != nil || cfg.Current == "" {
		return "Not signed in: every command that calls the API will ask you to run `auth signin`."
	}
	profile, err := cfg.Profile(cfg.Current)
	if err != nil {
		return "Not signed in: every command that calls the API will ask you to run `auth signin`."
	}
	who := profile.Caller
	if who == "" {
		who = "a token the server has not confirmed yet"
	}
	return fmt.Sprintf("Signed in to %s as %s (%s).", profile.APIURL, who, strings.Join(profile.Scopes, ","))
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
		Annotations: map[string]string{annoNoToken: "true"},
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
