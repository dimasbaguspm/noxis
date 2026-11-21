package commands

import (
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
				return fmt.Errorf("give the statement with --sql or --file")
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
			databases, err := api.Databases(cmd.Context(), ref)
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, databases)
			}
			for _, name := range databases {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), name); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

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
			tables, err := api.Tables(cmd.Context(), ref, args[1])
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, tables)
			}
			rows := make([][]string, 0, len(tables))
			for _, table := range tables {
				rows = append(rows, []string{
					table.Name, table.Engine, fmt.Sprint(table.Rows), fmt.Sprint(table.SizeBytes),
				})
			}
			return writeTable(cmd.OutOrStdout(), []string{"NAME", "ENGINE", "ROWS", "BYTES"}, rows)
		},
	}
}

func NewTable(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "table <ref|name> <schema> <table>",
		Short: "Show one table: columns, indexes and its DDL",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			ref, err := TargetRef(cmd.Context(), api, args[0])
			if err != nil {
				return err
			}
			detail, err := api.Table(cmd.Context(), ref, args[1], args[2])
			if err != nil {
				return err
			}
			if s.JSON {
				return PrintJSON(cmd, detail)
			}
			rows := make([][]string, 0, len(detail.Columns))
			for _, column := range detail.Columns {
				nullable := "not null"
				if column.Nullable {
					nullable = "null"
				}
				rows = append(rows, []string{column.Name, column.Type, nullable, column.Key, column.Extra})
			}
			if err := writeTable(cmd.OutOrStdout(), []string{"COLUMN", "TYPE", "NULL", "KEY", "EXTRA"}, rows); err != nil {
				return err
			}
			for _, index := range detail.Indexes {
				unique := ""
				if index.Unique {
					unique = " unique"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "index %s%s (%v)\n", index.Name, unique, index.Columns); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", detail.CreateSQL)
			return err
		},
	}
}
