package commands

import (
	"github.com/spf13/cobra"
)

// NewProposal is the one group a reviewer works in: every act on a change under one noun. The eight verbs
// it holds used to be eight top-level commands, which is eight names to remember before the first proposal
// is read.
//
// The old spellings still run — a script written against last month's binary is not a reason to break a
// deployment — but they are deprecated, hidden from help, and say where they went.
func NewProposal(s *Session) *cobra.Command {
	group := &cobra.Command{
		Use: "proposal",
		// `change` belongs to `propose`: the singular verb creates a change, the plural noun lists them.
		Aliases: []string{"changes"},
		Short:   "List, read, decide and apply a change",
		Long: "Every act on a proposal, under one noun:\n" +
			"  ls · show · rehearse · approve · reject · apply · watch · events\n\n" +
			"A proposal is decided by a different token holder than the one that proposed it.",
	}
	group.AddCommand(
		renamed(NewProposals(s), "ls", "List the change queue, newest first"),
		renamed(NewProposalShow(s), "show <ref|name>", "Show one proposal and the rehearsal that judged it"),
		renamed(NewRehearse(s), "rehearse <ref|name>", "Re-run the proof for a proposal nobody has decided yet"),
		renamed(NewApprove(s), "approve <ref|name>", "Approve a rehearsed proposal"),
		renamed(NewReject(s), "reject <ref|name>", "Reject a proposal"),
		renamed(NewApplyProposal(s), "apply <ref|name>", "Run an approved proposal against the target"),
		renamed(NewWatch(s), "watch <ref|name>", "Wait until a proposal settles, and print what it settled as"),
		renamed(NewEvents(s), "events <ref|name>", "Show the append-only history of one proposal"),
	)
	return group
}

// renamed gives a command the name it carries inside a group. `Use`, `Short` and the aliases are fields;
// the flags and the RunE are exactly what they were, so the behaviour cannot drift from the parent form.
func renamed(cmd *cobra.Command, use, short string) *cobra.Command {
	cmd.Use = use
	cmd.Short = short
	// A subcommand inherits its parent's context: an alias that made sense at the top level ("changes")
	// would shadow a sibling here.
	cmd.Aliases = nil
	return cmd
}

// Deprecated marks a top-level spelling that moved into a group. It still runs and still prints its own
// output; the notice goes to stderr, where a script's stdout is not disturbed.
func Deprecated(cmd *cobra.Command, use, replacement string) *cobra.Command {
	cmd.Use = use
	cmd.Aliases = nil // the group owns the aliases now
	cmd.Deprecated = "moved: run `noxis " + replacement + "`"
	return cmd
}
