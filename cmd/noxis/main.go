// Command noxis is the whole product in one binary: \`http serve\` is the API that owns the state
// file, \`web serve\` is the console, and every other word is the client that talks to the API over
// HTTP. One image, one entrypoint, no launcher script.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/client"
	"noxis/internal/common/dotconfig"
	"noxis/internal/constants"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Command annotations. A command that starts a session cannot require one, and a command that asks
// the server who it is must not be preceded by a second call asking the same question.
const (
	annoNoToken  = "no-token"
	annoNoVerify = "no-verify"
)

// options are the flags every command shares, and what boot resolved for them.
type options struct {
	apiURL  string
	token   string
	profile string
	asJSON  bool

	// resolved by prepare
	cfg        dotconfig.Config
	storedName string
	fromStore  bool
	name       string
	scopes     []string
}

func newRoot() *cobra.Command {
	opts := &options{}
	root := &cobra.Command{
		Use:           "noxis",
		Short:         "Client for the noXis API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&opts.apiURL, "api-url", "",
		"the noxisd to call (default $API_URL, then the signed-in profile)")
	root.PersistentFlags().StringVar(&opts.token, "token", "",
		"bearer token, for CI (default $API_TOKEN, then the signed-in profile)")
	root.PersistentFlags().StringVar(&opts.profile, "profile", "",
		"which stored profile to use (default the current one)")
	root.PersistentFlags().BoolVar(&opts.asJSON, "json", false, "print the response as json")

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if cmd.Annotations[annoNoToken] == "true" {
			return nil
		}
		return opts.prepare(cmd)
	}

	root.AddCommand(newHTTP(), newWeb(),
		newWhoami(opts), newVersion(), newAuth(opts), newTarget(opts),
		newQuery(opts), newDatabases(opts), newTables(opts), newTable(opts),
		newPropose(opts), newProposals(opts), newProposalShow(opts), newRehearse(opts),
		newApprove(opts), newReject(opts), newApplyProposal(opts), newEvents(opts))
	return root
}

// prepare decides which server and which token this command talks to, and proves the token is still
// one the server accepts. Precedence: the flag, then the stored profile, then the environment — the
// first is what a human typed now, the last is what CI sets and nothing else should.
func (o *options) prepare(cmd *cobra.Command) error {
	cfg, err := dotconfig.Load()
	if err != nil {
		return err
	}
	o.cfg = cfg

	name := o.profile
	if name == "" {
		name = cfg.Current
	}
	var stored dotconfig.Profile
	if name != "" {
		profile, err := cfg.Profile(name)
		if err != nil && o.profile != "" {
			// An explicitly named profile that is not there is a mistake, not a reason to fall back.
			return fmt.Errorf("%w: %q is not in %s", dotconfig.ErrNoProfile, name, cfg.FilePath())
		}
		if err == nil {
			stored, o.storedName = profile, name
		}
	}

	flagURL, flagToken := o.apiURL, o.token
	envToken := os.Getenv("API_TOKEN")
	o.apiURL = first(flagURL, stored.APIURL, os.Getenv("API_URL"), constants.DefaultAPIURL)
	o.token = first(flagToken, stored.Token, envToken)
	o.fromStore = flagToken == "" && envToken == "" && stored.Token != ""

	if o.token == "" {
		return fmt.Errorf("not signed in to %s: run `noxis auth signin <token> --api-url %s`",
			o.apiURL, o.apiURL)
	}
	if !o.fromStore {
		// A token in a shell variable is in the shell's history and in every child process. Say so
		// once, without ever printing the token itself.
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
			"noxis: using a token from --token or $API_TOKEN; `noxis auth signin` keeps it out of both")
		return nil
	}
	if cmd.Annotations[annoNoVerify] == "true" {
		return nil
	}
	return o.verify(cmd)
}

// verify asks the server who the stored token is, unless it asked recently. A token the server
// refuses is reported as what it is — sign in again — instead of as a 401 in the middle of a
// rehearsal nobody can explain.
func (o *options) verify(cmd *cobra.Command) error {
	stored, err := o.cfg.Profile(o.storedName)
	if err != nil {
		return nil
	}
	if stored.Caller != "" && time.Since(stored.CheckedAt) < constants.AuthCheckWindow {
		o.name, o.scopes = stored.Caller, stored.Scopes
		return nil
	}

	api, err := o.api()
	if err != nil {
		return err
	}
	identity, err := api.Whoami(cmd.Context())
	if err != nil {
		var refused *client.Error
		if errors.As(err, &refused) && refused.Status == http.StatusUnauthorized {
			return fmt.Errorf("the server at %s no longer accepts the token stored for %q: run `noxis auth signin <token> --api-url %s`",
				o.apiURL, o.storedName, o.apiURL)
		}
		return err
	}

	stored.Caller, stored.Scopes, stored.CheckedAt = identity.Name, identity.Scopes, time.Now().UTC()
	o.cfg.Put(o.storedName, stored)
	o.name, o.scopes = identity.Name, identity.Scopes
	// The cache is a courtesy: a config file that cannot be written must not fail the command the
	// human actually asked for.
	if err := o.cfg.Save(); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "noxis: the identity could not be cached: %v\n", err)
	}
	return nil
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

// first is the precedence rule in one line: the first value that is set wins.
func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
