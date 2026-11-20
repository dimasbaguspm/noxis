package main

import (
	"fmt"
	"noxis/internal/client"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newPropose(opts *options) *cobra.Command {
	var title, sqlText, from, rollback, rollbackFile, note, tier string
	cmd := &cobra.Command{
		Use:   "propose <target-ref>",
		Short: "Propose a change; it is rehearsed before you see it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			statement, err := content(sqlText, from)
			if err != nil {
				return err
			}
			if statement == "" {
				return fmt.Errorf("give the change with --sql or --file")
			}
			undo, err := content(rollback, rollbackFile)
			if err != nil {
				return err
			}
			api, err := opts.api()
			if err != nil {
				return err
			}
			created, err := api.CreateProposal(cmd.Context(), args[0], client.CreateProposalInput{
				Title: title, Query: statement, Rollback: undo, Note: note, Tier: tier,
			})
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, created)
			}
			return printProposal(cmd, created)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&title, "title", "", "what this change does, in one line")
	flags.StringVar(&sqlText, "sql", "", "the statement")
	flags.StringVar(&from, "file", "", "read the statement from this file")
	flags.StringVar(&rollback, "rollback", "", "the statement that undoes it")
	flags.StringVar(&rollbackFile, "rollback-file", "", "read the rollback from this file")
	flags.StringVar(&note, "note", "", "why, for the reviewer")
	flags.StringVar(&tier, "tier", "", "static, shadow (default) or rehearsal")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newProposals(opts *options) *cobra.Command {
	var limit int
	var cursor, status, kind, target string
	var onlyMine bool
	cmd := &cobra.Command{
		Use:   "proposals",
		Short: "List the proposal queue, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			page, err := api.ListProposals(cmd.Context(), limit, cursor, status, kind, target, onlyMine)
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, page)
			}
			rows := make([][]string, 0, len(page.Items))
			for _, item := range page.Items {
				rows = append(rows, []string{
					short(item.Ref), item.Kind, item.Status, item.Tier,
					irreversibleMark(item.Irreversible), item.Title,
				})
			}
			if err := writeTable(cmd.OutOrStdout(),
				[]string{"REF", "KIND", "STATUS", "TIER", "", "TITLE"}, rows); err != nil {
				return err
			}
			if page.NextCursor != "" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nnext: --cursor %s\n", page.NextCursor)
			}
			return err
		},
	}
	flags := cmd.Flags()
	flags.IntVar(&limit, "limit", 0, "page size, default 50")
	flags.StringVar(&cursor, "cursor", "", "cursor from the previous page")
	flags.StringVar(&status, "status", "", "only this status")
	flags.StringVar(&kind, "kind", "", "dml or ddl")
	flags.StringVar(&target, "target", "", "only this target")
	flags.BoolVar(&onlyMine, "mine", false, "only the proposals this token made")
	return cmd
}

func newProposalShow(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "show-proposal <ref>",
		Short: "Show one proposal and the rehearsal that judged it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			item, err := api.GetProposal(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, item)
			}
			return printProposal(cmd, item)
		},
	}
}

func newRehearse(opts *options) *cobra.Command {
	var tier string
	cmd := &cobra.Command{
		Use:   "rehearse <proposal-ref>",
		Short: "Re-run the proof for a proposal nobody has decided yet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			updated, err := api.RehearseProposal(cmd.Context(), args[0], tier)
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, updated)
			}
			return printProposal(cmd, updated)
		},
	}
	cmd.Flags().StringVar(&tier, "tier", "", "static, shadow or rehearsal")
	return cmd
}

// printProposal prints the change, its verdict and the evidence. A reviewer decides from this, so
// the diffs are printed whole and never summarised.
func printProposal(cmd *cobra.Command, proposal client.Proposal) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "%s  %s\n%s\n\n", short(proposal.Ref), proposal.Status, proposal.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "kind %s · tier %s · proposed by %s%s\n\n",
		proposal.Kind, proposal.Tier, proposal.ProposedBy, irreversibleNote(proposal.Irreversible)); err != nil {
		return err
	}
	if err := printSQL(out, "change", proposal.Query); err != nil {
		return err
	}
	if err := printSQL(out, "rollback", proposal.Rollback); err != nil {
		return err
	}
	if proposal.Failure != "" {
		if _, err := fmt.Fprintf(out, "failure:\n%s\n", proposal.Failure); err != nil {
			return err
		}
	}
	if proposal.Rehearsal == nil {
		return nil
	}
	rehearsal := proposal.Rehearsal
	if _, err := fmt.Fprintf(out, "\nrehearsal %s · %s · %d ms\n%s\n",
		rehearsal.Status, rehearsal.Tier, rehearsal.DurationMS, rehearsal.Note); err != nil {
		return err
	}
	if rehearsal.Error != "" {
		if _, err := fmt.Fprintf(out, "error: %s\n", rehearsal.Error); err != nil {
			return err
		}
	}
	for _, section := range []struct{ name, body string }{
		{"schema diff", rehearsal.SchemaDiff},
		{"row diff", rehearsal.RowDiff},
		{"plan", rehearsal.Plan},
	} {
		if strings.TrimSpace(section.body) == "" {
			continue
		}
		if _, err := fmt.Fprintf(out, "\n%s:\n%s", section.name, indent(section.body)); err != nil {
			return err
		}
	}
	return nil
}

func printSQL(out interface{ Write([]byte) (int, error) }, label, statement string) error {
	if strings.TrimSpace(statement) == "" {
		_, err := fmt.Fprintf(out, "%s: (none)\n", label)
		return err
	}
	_, err := fmt.Fprintf(out, "%s:\n%s", label, indent(statement))
	return err
}

func indent(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n") + "\n"
}

// short keeps a ref readable in a table without losing the part that tells two apart: the timestamp
// prefix of a UUIDv7 is identical for everything made in the same millisecond.
func short(ref string) string {
	const keep = 18
	if len(ref) > keep {
		return ref[:keep]
	}
	return ref
}

func irreversibleMark(irreversible bool) string {
	if irreversible {
		return "irreversible"
	}
	return ""
}

func irreversibleNote(irreversible bool) string {
	if irreversible {
		return " · irreversible: no rollback was given"
	}
	return ""
}

// content reads a statement from the flag or from a file, so a multi-line change does not have to
// be quoted on a command line.
func content(inline, file string) (string, error) {
	if file == "" {
		return inline, nil
	}
	raw, err := os.ReadFile(file) //nolint:gosec // the caller names the file on purpose
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func newApprove(opts *options) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "approve <proposal-ref>",
		Short: "Approve a rehearsed proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return decide(cmd, opts, args[0], note, true)
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "why, for the record")
	return cmd
}

func newReject(opts *options) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "reject <proposal-ref>",
		Short: "Reject a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return decide(cmd, opts, args[0], note, false)
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "why, for the record")
	return cmd
}

func decide(cmd *cobra.Command, opts *options, ref, note string, approve bool) error {
	api, err := opts.api()
	if err != nil {
		return err
	}
	var updated client.Proposal
	if approve {
		updated, err = api.ApproveProposal(cmd.Context(), ref, note)
	} else {
		updated, err = api.RejectProposal(cmd.Context(), ref, note)
	}
	if err != nil {
		return err
	}
	if opts.asJSON {
		return printJSON(cmd, updated)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", short(updated.Ref), updated.Status)
	return err
}

func newApplyProposal(opts *options) *cobra.Command {
	var confirm bool
	cmd := &cobra.Command{
		Use:   "apply <proposal-ref>",
		Short: "Run an approved proposal against the target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			applied, err := api.ApplyProposal(cmd.Context(), args[0], confirm)
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, applied)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s by %s\n", short(applied.Ref), applied.Status, applied.AppliedBy)
			return err
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm-destructive", false, "acknowledge that the change cannot be undone")
	return cmd
}

func newEvents(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "events <proposal-ref>",
		Short: "Show the append-only history of one proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := opts.api()
			if err != nil {
				return err
			}
			events, err := api.ProposalEvents(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if opts.asJSON {
				return printJSON(cmd, events)
			}
			rows := make([][]string, 0, len(events))
			for _, event := range events {
				rows = append(rows, []string{event.CreatedAt.Format("15:04:05"), event.Kind, event.Actor})
			}
			return writeTable(cmd.OutOrStdout(), []string{"AT", "KIND", "ACTOR"}, rows)
		},
	}
}
