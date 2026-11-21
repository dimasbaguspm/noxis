package main

import (
	"fmt"
	"noxis/cmd/noxis/commands"
	"noxis/internal/constants"
	"noxis/skills"

	"github.com/spf13/cobra"
)

// newSkills prints the agent skills this build carries. They are embedded rather than fetched, because the
// instructions an agent follows and the binary that answers it have to be the same version: a skill from a
// newer release describes flags this CLI does not have, and the agent has no way to notice.
func newSkills(s *commands.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills [name]",
		Short: "Print the agent skills that ship with this build",
		Long: "With no argument: every skill this build carries, one line each. With a name: the document\n" +
			"whole, ready to redirect into an agent's skills directory.\n\n" +
			"  noxis skills noxis-read > ~/.claude/skills/noxis-read/SKILL.md",
		Args:        cobra.MaximumNArgs(1),
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				body, err := skills.Read(args[0])
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), body)
				return err
			}

			all, err := skills.List()
			if err != nil {
				return err
			}
			if s.JSON {
				return commands.PrintJSON(cmd, all)
			}
			for _, skill := range all {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-14s %s\n", skill.Name, skill.Description); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"\nthis build: %s\nread one:   noxis skills <name>\ninstall:    noxis skills <name> > <agent>/skills/<name>/SKILL.md\n",
				constants.Build())
			return err
		},
	}
	return cmd
}
