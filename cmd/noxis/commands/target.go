package commands

import (
	"bytes"
	"fmt"
	"io"
	"noxis/internal/client"
	"noxis/internal/constants"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func NewTarget(s *Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "target",
		Aliases: []string{"instance", "instances", "targets"},
		Short:   "Manage the instances this noxis governs",
	}
	cmd.AddCommand(newTargetAdd(s), newTargetList(s), newTargetShow(s), newTargetProbe(s),
		newTargetTest(s), newTargetSet(s), newTargetArchive(s))
	return cmd
}

func newTargetAdd(s *Session) *cobra.Command {
	in := client.CreateTargetInput{}
	cmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"create", "new"},
		Short:   "Register a MySQL or MariaDB instance",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			created, err := api.CreateTarget(cmd.Context(), in)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, created)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), created.Ref)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&in.Name, "name", "", "how this target is named in noxis")
	flags.StringVar(&in.Driver, "driver", constants.DriverMySQL, "mysql or mariadb")
	flags.StringVar(&in.Host, "host", "", "host to connect to")
	flags.IntVar(&in.Port, "port", 0, "port, default 3306")
	flags.StringVar(&in.Database, "database", "", "database to govern")
	flags.StringVar(&in.Username, "username", "", "user to connect as")
	flags.StringVar(&in.Password, "password", "", "password, which lands in the shell history: prefer the API with a secret store")
	flags.StringVar(&in.TLSMode, "tls-mode", constants.TLSPreferred, "preferred, skip-verify or verify")
	flags.StringVar(&in.IdempotencyKey, "idempotency-key", "",
		"a name for this registration; a retry with the same one answers the same instance")
	for _, required := range []string{"name", "host", "database", "username"} {
		_ = cmd.MarkFlagRequired(required)
	}
	return cmd
}

func newTargetList(s *Session) *cobra.Command {
	in := client.ListTargetsInput{}
	var everyPage bool
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List instances, newest first",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			page, err := api.ListTargets(cmd.Context(), in)
			if err != nil {
				return err
			}
			if everyPage {
				page, err = walkAll(page, func(cursor string) (client.Page[client.Target], error) {
					next := in
					next.Cursor = cursor
					return api.ListTargets(cmd.Context(), next)
				})
				if err != nil {
					return err
				}
			}
			if s.JSON {
				return PrintJSON(cmd, page)
			}
			rows := make([][]string, 0, len(page.Items))
			for _, t := range page.Items {
				rows = append(rows, []string{
					t.Name, t.Driver, fmt.Sprintf("%s:%d", t.Host, t.Port), t.Database, t.Ref,
				})
			}
			if err := writeTable(cmd.OutOrStdout(),
				[]string{"NAME", "DRIVER", "ADDRESS", "DATABASE", "REF"}, rows); err != nil {
				return err
			}
			if page.NextCursor != "" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nnext: --cursor %s\n", page.NextCursor)
			}
			return err
		},
	}
	flags := cmd.Flags()
	flags.IntVar(&in.Limit, "limit", 0, "page size, default 50")
	flags.StringVar(&in.Cursor, "cursor", "", "cursor from the previous page")
	flags.BoolVar(&everyPage, "all", false, "every page, not just the first")
	flags.StringVar(&in.Driver, "driver", "", "only this driver")
	flags.StringVar(&in.Query, "q", "", "filter by name")
	flags.BoolVar(&in.IncludeArchived, "archived", false, "include archived instances")
	return cmd
}

func newTargetShow(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:     "show <ref>",
		Aliases: []string{"get"},
		Short:   "Show one instance",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			target, err := api.GetTarget(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, target)
			}
			rows := [][]string{
				{"ref", target.Ref},
				{"name", target.Name},
				{"driver", target.Driver},
				{"address", fmt.Sprintf("%s:%d", target.Host, target.Port)},
				{"database", target.Database},
				{"username", target.Username},
				{"tls", target.TLSMode},
				{"password", passwordState(target.HasPassword)},
				{"engine", engineLine(target)},
				{"probe", probeLine(target)},
				{"created", fmt.Sprintf("%s by %s", target.CreatedAt.Format("2006-01-02 15:04"), target.CreatedBy)},
			}
			return writeTable(cmd.OutOrStdout(), nil, rows)
		},
	}
}

func newTargetArchive(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <ref>",
		Short: "Archive a target. Nothing is deleted",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			archived, err := api.ArchiveTarget(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, archived)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s archived\n", archived.Name)
			return err
		},
	}
}

func passwordState(has bool) string {
	if has {
		return "set"
	}
	return "none"
}

// writeTable renders rows into a buffer and writes the result once, so a broken pipe is
// reported rather than swallowed by a dozen unchecked prints. Writes into the buffer
// cannot fail, which is why only the single output write is checked.
func writeTable(w io.Writer, header []string, rows [][]string) error {
	var buf bytes.Buffer
	table := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	if len(header) > 0 {
		_, _ = fmt.Fprintln(table, strings.Join(header, "\t"))
	}
	for _, row := range rows {
		_, _ = fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	if err := table.Flush(); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func newTargetProbe(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "probe <ref>",
		Short: "Connect once and record what the engine actually is",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			result, err := api.ProbeTarget(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, result)
			}
			return printCapabilities(cmd, result.Capabilities)
		},
	}
}

func engineLine(target client.Target) string {
	if target.ServerVersion == "" {
		return target.Driver + " (declared, not probed)"
	}
	return target.Driver + " · " + target.ServerVersion
}

func probeLine(target client.Target) string {
	if target.LastProbeStatus == "" {
		return "never"
	}
	if target.LastProbeAt == nil {
		return target.LastProbeStatus
	}
	return target.LastProbeStatus + " at " + target.LastProbeAt.Format("2006-01-02 15:04")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// printCapabilities is one answer for a probe and for a test: what the engine said it is, and what
// this build may therefore do with it.
func printCapabilities(cmd *cobra.Command, caps client.Capabilities) error {
	rows := make([][]string, 0, 5+len(caps.Notes))
	rows = append(rows,
		[]string{"engine", caps.Flavor},
		[]string{"version", caps.Version},
		[]string{"explain json", yesNo(caps.ExplainJSON)},
		[]string{"shadow ddl rehearsal", yesNo(caps.ShadowDDLRehearsal)},
		[]string{"row image rehearsal", yesNo(caps.RowImageRehearsal)},
	)
	for _, note := range caps.Notes {
		rows = append(rows, []string{"note", note})
	}
	return writeTable(cmd.OutOrStdout(), nil, rows)
}

// newTargetTest dials a configuration nobody has stored — the console's "Test connection", from the
// CLI. It writes nothing, so a typo costs a connection rather than a row somebody has to archive.
func newTargetTest(s *Session) *cobra.Command {
	in := client.TestTargetInput{}
	cmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"check"},
		Short:   "Dial the credentials as typed, and store nothing",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			result, err := api.TestTarget(cmd.Context(), in)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, result)
			}
			if err := printCapabilities(cmd, result.Capabilities); err != nil {
				return err
			}
			if len(result.Databases) == 0 {
				return nil
			}
			return writeTable(cmd.OutOrStdout(), nil,
				[][]string{{"schemas", strings.Join(result.Databases, ", ")}})
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&in.Driver, "driver", constants.DriverMySQL, "mysql or mariadb")
	flags.StringVar(&in.Host, "host", "", "host to connect to")
	flags.IntVar(&in.Port, "port", 0, "port, default 3306")
	flags.StringVar(&in.Database, "database", "", "database to connect to")
	flags.StringVar(&in.Username, "username", "", "user to connect as")
	flags.StringVar(&in.Password, "password", "", "password, which lands in the shell history: prefer the API with a secret store")
	flags.StringVar(&in.TLSMode, "tls-mode", constants.TLSPreferred, "preferred, skip-verify or verify")
	return cmd
}

// newTargetSet corrects a connection. With --probe it dials the new credentials first, the same dial a
// registration makes: a correction nobody proved is a proposal that fails its rehearsal an hour later.
func newTargetSet(s *Session) *cobra.Command {
	var name, username, password, tlsMode string
	var probe bool
	cmd := &cobra.Command{
		Use:     "set <ref>",
		Aliases: []string{"edit", "update"},
		Short:   "Change a connection, and prove the new credentials before saving them",
		Long: "Changes the fields you name and leaves the rest alone. --probe dials the new credentials\n" +
			"without storing anything, so a wrong password is a failed dial rather than a failed rehearsal.\n" +
			"A probe needs --password: the stored one is sealed and never read back to be typed again.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			in := client.UpdateTargetInput{}
			// Only the fields the caller named travel: an unset flag is "leave it as it is", which is what
			// a pointer field in the update input means.
			if name != "" {
				in.Name = &name
			}
			if username != "" {
				in.Username = &username
			}
			if password != "" {
				in.Password = &password
			}
			if tlsMode != "" {
				in.TLSMode = &tlsMode
			}
			if in.Name == nil && in.Username == nil && in.Password == nil && in.TLSMode == nil {
				return fmt.Errorf("name a field to change: --name, --username, --password or --tls-mode")
			}
			if probe {
				if password == "" {
					return fmt.Errorf("--probe needs --password: the stored one is never read back")
				}
				current, err := api.GetTarget(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				proved, err := api.TestTarget(cmd.Context(), client.TestTargetInput{
					Driver:   current.Driver,
					Host:     current.Host,
					Port:     current.Port,
					Database: current.Database,
					Username: first(username, current.Username),
					Password: password,
					TLSMode:  first(tlsMode, current.TLSMode),
					Options:  current.Options,
				})
				if err != nil {
					return fmt.Errorf("the new credentials were not proved, so nothing was saved: %w", err)
				}
				if !s.JSON {
					if err := printCapabilities(cmd, proved.Capabilities); err != nil {
						return err
					}
				}
			}
			updated, err := api.UpdateTarget(cmd.Context(), args[0], in)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, updated)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s saved\n", updated.Ref, updated.Name)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "the new name; leave it unset to keep the old one")
	flags.StringVar(&username, "username", "", "the new user; leave it unset to keep the old one")
	flags.StringVar(&password, "password", "", "the new password, which lands in the shell history")
	flags.StringVar(&tlsMode, "tls-mode", "", "preferred, skip-verify or verify")
	flags.BoolVar(&probe, "probe", false, "dial the new credentials before saving them")
	return cmd
}
