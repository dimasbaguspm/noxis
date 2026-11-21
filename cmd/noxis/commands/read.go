package commands

import (
	"errors"
	"fmt"
	"noxis/internal/client"
	"os"

	"github.com/spf13/cobra"
)

// NewQuery runs one statement. The server refuses anything that is not a read, so this command
// does not decide what a read is: it sends and prints, and the refusal comes back as an error.
func NewQuery(s *Session) *cobra.Command {
	var (
		sqlText string
		from    string
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "query <ref|name>",
		Short: "Run a SELECT, EXPLAIN or SHOW against a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			statement := sqlText
			if from != "" {
				content, err := os.ReadFile(from) //nolint:gosec // the caller names the file on purpose
				if err != nil {
					return err
				}
				statement = string(content)
			}
			if statement == "" {
				return errors.New("give the statement with --sql or --file")
			}
			api, err := s.Client()
			if err != nil {
				return err
			}
			ref, err := TargetRef(cmd.Context(), api, args[0])
			if err != nil {
				return err
			}
			rows, err := api.Query(cmd.Context(), ref, statement, limit)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, rows)
			}
			return printRows(cmd, rows)
		},
	}
	cmd.Flags().StringVar(&sqlText, "sql", "", "the statement to run")
	cmd.Flags().StringVar(&from, "file", "", "read the statement from this file")
	cmd.Flags().IntVar(&limit, "limit", 0, "row ceiling, default 200, max 1000")
	return cmd
}

func printRows(cmd *cobra.Command, rows client.Rows) error {
	header := make([]string, 0, len(rows.Columns))
	for _, column := range rows.Columns {
		header = append(header, column.Name)
	}
	body := make([][]string, 0, len(rows.Values))
	for _, row := range rows.Values {
		line := make([]string, 0, len(row))
		for _, value := range row {
			if value == nil {
				line = append(line, "NULL")
				continue
			}
			line = append(line, *value)
		}
		body = append(body, line)
	}
	if err := writeTable(cmd.OutOrStdout(), header, body); err != nil {
		return err
	}
	note := fmt.Sprintf("%d row(s) in %d ms", rows.RowCount, rows.DurationMS)
	if rows.Truncated {
		note += " — truncated: raise --limit or narrow the statement"
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), note)
	return err
}

// NewDatabases is the old spelling of `noxis table <target>`: one depth of one question, kept at the root
// so a script runs, deprecated so nobody new learns it.
func NewDatabases(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "databases <ref|name>",
		Short: "List the schemas behind a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			ref, err := TargetRef(cmd.Context(), api, args[0])
			if err != nil {
				return err
			}
			return printDatabases(cmd, s, api, ref)
		},
	}
}

// NewTables is the old spelling of `noxis table <target> <schema>`.
func NewTables(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "tables <ref|name> <schema>",
		Short: "List the tables in a schema",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			ref, err := TargetRef(cmd.Context(), api, args[0])
			if err != nil {
				return err
			}
			return printTables(cmd, s, api, ref, args[1])
		},
	}
}
