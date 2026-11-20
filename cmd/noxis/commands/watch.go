package commands

import (
	"context"
	"fmt"
	"noxis/internal/client"
	"noxis/internal/constants"
	"time"

	"github.com/spf13/cobra"
)

// settledStatuses is the default `--until`: the statuses a proposal does not leave by itself. A
// caller may name fewer — `--until applied` waits for the good ending only — and naming a status the
// machine never rests in is a wait with no end, so the default is all three.
var settledStatuses = []string{"applied", "failed", "rejected"}

// applying is the status apply answers with while the statement runs in the background. It is the
// only status a caller has to wait past, which is why `apply --wait` exists at all.
const applying = "applying"

func NewWatch(s *Session) *cobra.Command {
	var until []string
	var interval, timeout time.Duration
	cmd := &cobra.Command{
		Use:   "watch <proposal-ref>",
		Short: "Wait until a proposal settles, and print what it settled as",
		Long: "Asks the API what a proposal is doing until it reaches a status it does not leave.\n\n" +
			"A change applied above " + fmt.Sprint(constants.ApplyBackgroundRows) + " rows is answered with a running\n" +
			"status and finishes later; this is how a caller waits for the ending instead of polling by\n" +
			"hand. The exit code is zero only when one of the named statuses arrived.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(until) == 0 {
				return fmt.Errorf("--until needs at least one status, for example applied")
			}
			api, err := s.Client()
			if err != nil {
				return err
			}
			proposal, err := waitFor(cmd.Context(), api, args[0], until, interval, timeout)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, proposal)
			}
			return printWatched(cmd, proposal)
		},
	}
	flags := cmd.Flags()
	flags.StringSliceVar(&until, "until", settledStatuses, "the statuses that end the wait")
	flags.DurationVar(&interval, "interval", constants.WatchInterval, "how often to ask")
	flags.DurationVar(&timeout, "timeout", constants.WatchTimeout, "how long to wait before giving up")
	return cmd
}

// waitFor asks until a named status arrives or the deadline passes. The first ask happens before the
// first tick, so a proposal that already settled answers at once instead of one interval later.
func waitFor(ctx context.Context, api *client.Client, proposalRef string, until []string,
	interval, timeout time.Duration,
) (client.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	last := client.Proposal{}
	for {
		proposal, err := api.GetProposal(ctx, proposalRef)
		if err != nil {
			// The ask itself can be the thing the deadline cuts: a request stopped mid-flight is not a
			// different failure from a wait that ran out, so it is reported as the wait it was.
			if ctx.Err() != nil {
				return last, gaveUp(last, timeout)
			}
			return client.Proposal{}, err
		}
		last = proposal
		if settled(proposal.Status, until) {
			return proposal, nil
		}
		select {
		case <-ctx.Done():
			return proposal, gaveUp(proposal, timeout)
		case <-ticker.C:
		}
	}
}

func settled(status string, until []string) bool {
	for _, candidate := range until {
		if candidate == status {
			return true
		}
	}
	return false
}

// gaveUp says what the wait was watching and how long it waited: a timeout a caller can act on names
// the status it was stuck in, and says how to wait longer.
func gaveUp(proposal client.Proposal, timeout time.Duration) error {
	if proposal.Ref == "" {
		return fmt.Errorf("no answer within %s: the API did not respond before the deadline", timeout)
	}
	return fmt.Errorf("%s was still %q after %s: raise --timeout, or read the journal for what the apply is doing",
		short(proposal.Ref), proposal.Status, timeout)
}

// printWatched says what the wait was for: the status it settled as, and the failure when the
// verdict was one. The diffs belong to `show-proposal`, not to a line a script greps.
func printWatched(cmd *cobra.Command, proposal client.Proposal) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "%s  %s\n", short(proposal.Ref), proposal.Status); err != nil {
		return err
	}
	if proposal.Failure == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "failure:\n%s", indent(proposal.Failure))
	return err
}
