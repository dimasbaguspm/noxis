package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"noxis/internal/constants"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// operation is one line of the API as this token sees it: what the route does, the scopes it needs,
// and whether the token in hand holds every one of them.
type operation struct {
	Method  string   `json:"method"`
	Path    string   `json:"path"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags,omitempty"`
	Scopes  []string `json:"scopes"`
	Allowed bool     `json:"allowed"`
}

// description is what an agent asks for first: the API this server serves, joined with what this
// token may call, so a plan is made against the truth instead of against a failed request.
type description struct {
	Binary     string      `json:"binary"`
	APIURL     string      `json:"api_url"`
	APIVersion string      `json:"api_version,omitempty"`
	Caller     string      `json:"caller,omitempty"`
	Scopes     []string    `json:"scopes,omitempty"`
	Tiers      []string    `json:"tiers"`
	ErrorCodes []string    `json:"error_codes"`
	Operations []operation `json:"operations"`
}

// methods are the keys of a path item that name an operation. `parameters` lives beside them and is
// not one, which is why the document is walked method by method rather than decoded whole.
var methods = []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodPut}

// errorCodes is the vocabulary a failure can arrive in, taken from the constants so a rename cannot
// leave this list describing a code the server no longer sends.
var errorCodes = []string{
	constants.CodeInvalidRequest,
	constants.CodeUnauthorized,
	constants.CodeForbidden,
	constants.CodeNotFound,
	constants.CodeConflict,
	constants.CodeUnsupported,
	constants.CodeRehearsalFailed,
	constants.CodeClient,
	constants.CodeInternal,
}

// NewDescribe answers what this deployment offers and what this token may call. It reads the document
// the server serves rather than a copy embedded in the binary: a describe against a stale deployment
// describes that deployment, which is the only useful answer.
func NewDescribe(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "describe",
		Short: "The API this server serves, and which of it this token may call",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			// Whoami rather than the session's cached scopes: a token passed as a flag is never cached, and
			// describe is precisely the question "what may this token call".
			identity, err := api.Whoami(cmd.Context())
			if err != nil {
				return err
			}
			raw, err := api.OpenAPI(cmd.Context())
			if err != nil {
				return err
			}
			operations, version, err := operationsOf(raw, identity.Scopes)
			if err != nil {
				return err
			}
			described := description{
				Binary:     constants.Version,
				APIURL:     s.APIURL,
				APIVersion: version,
				Caller:     identity.Name,
				Scopes:     identity.Scopes,
				Tiers:      constants.Tiers,
				ErrorCodes: errorCodes,
				Operations: operations,
			}
			if s.JSON {
				return PrintJSON(cmd, described)
			}
			return printDescription(cmd, described)
		},
	}
}

// operationsOf walks the document and answers each operation with the scopes the route table declared,
// marked callable when the token holds all of them. A route with no scopes needs no scope, which is
// the same rule the auth middleware applies.
func operationsOf(raw []byte, tokenScopes []string) ([]operation, string, error) {
	var document struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, "", fmt.Errorf("the document this server serves does not parse: %w", err)
	}

	operations := make([]operation, 0, len(document.Paths))
	for path, item := range document.Paths {
		for method, body := range item {
			name := strings.ToUpper(method)
			if !slices.Contains(methods, name) {
				continue
			}
			var detail struct {
				Summary string   `json:"summary"`
				Tags    []string `json:"tags"`
				Scopes  []string `json:"x-scopes"`
			}
			if err := json.Unmarshal(body, &detail); err != nil {
				return nil, "", fmt.Errorf("the document's %s %s does not parse: %w", name, path, err)
			}
			operations = append(operations, operation{
				Method:  name,
				Path:    path,
				Summary: detail.Summary,
				Tags:    detail.Tags,
				Scopes:  detail.Scopes,
				Allowed: holds(tokenScopes, detail.Scopes),
			})
		}
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Path != operations[j].Path {
			return operations[i].Path < operations[j].Path
		}
		return operations[i].Method < operations[j].Method
	})
	return operations, document.Info.Version, nil
}

// holds reports whether the token holds every scope a route needs. No scopes needed is no scope
// required: such a route is callable by this token too.
func holds(tokenScopes, needed []string) bool {
	for _, scope := range needed {
		if !slices.Contains(tokenScopes, scope) {
			return false
		}
	}
	return true
}

// printDescription is the human shape: where this points, who it is, and the table of what is callable
// — the same document `--json` answers, cut down to a screen somebody reads once.
func printDescription(cmd *cobra.Command, described description) error {
	callable := 0
	for _, op := range described.Operations {
		if op.Allowed {
			callable++
		}
	}
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "noxis %s · %s · %s (%s)\n", described.Binary, described.APIURL,
		described.Caller, strings.Join(described.Scopes, ",")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%d of %d operations callable with this token\n\n",
		callable, len(described.Operations)); err != nil {
		return err
	}

	rows := make([][]string, 0, len(described.Operations))
	for _, op := range described.Operations {
		scopes := strings.Join(op.Scopes, ",")
		if scopes == "" {
			scopes = "none"
		}
		allowed := "no"
		if op.Allowed {
			allowed = "yes"
		}
		rows = append(rows, []string{op.Method, op.Path, scopes, allowed, op.Summary})
	}
	if err := writeTable(out, []string{"METHOD", "PATH", "SCOPES", "CALLABLE", "WHAT IT DOES"}, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\ntiers: %s\nerror codes: %s\n",
		strings.Join(described.Tiers, ", "), strings.Join(described.ErrorCodes, ", "))
	return err
}
