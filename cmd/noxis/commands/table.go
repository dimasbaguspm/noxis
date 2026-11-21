package commands

import (
	"fmt"
	"noxis/internal/client"
	"strconv"

	"github.com/spf13/cobra"
)

// NewTable is one command for schema metadata, because it is one question at three depths: which schemas
// exist, which tables are in this one, and what is inside this table. The argument list says the depth, so
// there is nothing to remember beyond how specific you want to be.
func NewTable(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "table <ref|name> [schema] [table]",
		Short: "List schemas, list tables, or show one table",
		Long: "Three depths, one command:\n" +
			"  table <target>                 every schema those credentials can see\n" +
			"  table <target> <schema>        the tables in it\n" +
			"  table <target> <schema> <name> columns, indexes and the DDL\n\n" +
			"`tables` reports an estimate from information_schema, not a count: a count costs what a reviewer\n" +
			"is trying to avoid.",
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := s.Client()
			if err != nil {
				return err
			}
			ref, err := TargetRef(cmd.Context(), api, args[0])
			if err != nil {
				return err
			}
			switch len(args) {
			case 1:
				return printDatabases(cmd, s, api, ref)
			case 2:
				return printTables(cmd, s, api, ref, args[1])
			default:
				return printTable(cmd, s, api, ref, args[1], args[2])
			}
		},
	}
}

// printDatabases is the first depth: every schema the target's credentials can see.
func printDatabases(cmd *cobra.Command, s *Session, api *client.Client, ref string) error {
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
}

// printTables is the second depth: the tables in one schema, with the estimates the server reads.
func printTables(cmd *cobra.Command, s *Session, api *client.Client, ref, schema string) error {
	tables, err := api.Tables(cmd.Context(), ref, schema)
	if err != nil {
		return err
	}
	if s.JSON {
		return PrintJSON(cmd, tables)
	}
	rows := make([][]string, 0, len(tables))
	for _, table := range tables {
		rows = append(rows, []string{
			table.Name, table.Engine, strconv.FormatInt(table.Rows, 10), strconv.FormatInt(table.SizeBytes, 10),
		})
	}
	return writeTable(cmd.OutOrStdout(), []string{"NAME", "ENGINE", "ROWS", "BYTES"}, rows)
}

// printTable is the third depth: columns, indexes, and the engine's own DDL.
func printTable(cmd *cobra.Command, s *Session, api *client.Client, ref, schema, table string) error {
	detail, err := api.Table(cmd.Context(), ref, schema, table)
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
}
