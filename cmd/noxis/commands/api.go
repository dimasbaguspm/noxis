package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// NewAPI is the contract and a way to call it. `describe` answers what this server serves and what the
// token may call; `get` and `post` reach any route the CLI does not model yet.
//
// It carries no privileges of its own: the token's scopes are the whole permission, so `api post` can do
// nothing a narrower command could not. A write needs `--yes`, because the route's name is not a warning.
func NewAPI(s *Session) *cobra.Command {
	group := &cobra.Command{
		Use:   "api",
		Short: "The contract this server serves, and a way to call it",
		Long: "The escape hatch, for a route this CLI does not model yet:\n" +
			"  api describe                          what this server serves, and what your token may call\n" +
			"  api get /v1/proposals?status=verified the envelope, exactly as the server wrote it\n" +
			"  api post /v1/targets/<ref>/probe --yes\n\n" +
			"A write needs --yes. The token's scopes decide what is allowed, not this command.",
	}
	group.AddCommand(
		renamed(NewDescribe(s), "describe", "The API this server serves, and which of it this token may call"),
		newAPICall(s, http.MethodGet),
		newAPICall(s, http.MethodPost),
	)
	return group
}

func newAPICall(s *Session, method string) *cobra.Command {
	var data string
	var yes bool
	name := strings.ToLower(method)

	cmd := &cobra.Command{
		Use:   name + " <path>",
		Short: strings.ToUpper(name) + " any route and print the envelope",
		Long: fmt.Sprintf("%s any route on this server and print the answer exactly as it came. The path is\n"+
			"relative to the API: `/v1/proposals`, `/v1/targets/<ref>/capabilities`.", strings.ToUpper(name)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			// The client prefixes its own base URL: a path that names a host would be a second destination
			// this command was never asked to trust.
			if !strings.HasPrefix(path, "/") || strings.Contains(path, "://") {
				return errors.New("the path starts with / and names a route, not a host")
			}
			if method != http.MethodGet && !yes {
				return fmt.Errorf("%s changes something on this server: add --yes", strings.ToUpper(name))
			}

			var body any
			if data != "" {
				decoded := map[string]any{}
				if err := json.Unmarshal([]byte(data), &decoded); err != nil {
					return fmt.Errorf("--data is not a json object: %w", err)
				}
				body = decoded
			}

			api, err := s.Client()
			if err != nil {
				return err
			}
			raw, err := api.Raw(cmd.Context(), method, path, body)
			if err != nil {
				return err
			}
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, raw, "", "  "); err != nil {
				// Not JSON? Print it as it came rather than pretend it cannot happen.
				_, err := cmd.OutOrStdout().Write(raw)
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), pretty.String())
			return err
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "a JSON object to send as the body")
	if method != http.MethodGet {
		cmd.Flags().BoolVar(&yes, "yes", false, "confirm a call that changes something")
	}
	return cmd
}
