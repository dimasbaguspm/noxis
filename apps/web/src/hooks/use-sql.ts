import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// Every hook here takes the driver of the connection it reads: the API addresses an engine by driver, so "postgres" and "sqlite" are different paths even though the console groups them together.

// useConnectionSchemaQuery returns the tables of one SQL connection; the completion and table browser read the same cached list.
export function useConnectionSchemaQuery(driver?: string, connection?: string) {
  return useQuery({
    queryKey: queryKeys.sql.schema(connection),
    queryFn: () => api.connections(driver as string).schema(connection as string),
    enabled: !!driver && !!connection,
    select: (res) => res.tables,
  });
}

// useErdQuery loads one connection's schema graph; the ERD page is the only caller, so it runs whenever a connection is picked.
export function useErdQuery(driver?: string, connection?: string) {
  return useQuery({
    queryKey: queryKeys.sql.erd(connection),
    queryFn: () => api.connections(driver as string).erd(connection as string),
    enabled: !!driver && !!connection,
  });
}

// useTableRowsQuery previews a table's rows; enabled only once a table is picked, so the cache re-runs when the selection changes.
export function useTableRowsQuery(
  driver?: string,
  connection?: string,
  schema?: string,
  table?: string,
) {
  return useQuery({
    queryKey: queryKeys.sql.rows(connection, schema, table),
    queryFn: () =>
      api
        .connections(driver as string)
        .rows(connection as string, schema as string, table as string),
    enabled: !!driver && !!connection && !!schema && !!table,
  });
}

// useRunQueryMutation executes a read against the instance; the result is the mutation's data, so a fresh run replaces the last result.
export function useRunQueryMutation() {
  return useMutation({
    mutationFn: ({
      driver,
      connection,
      sql,
    }: {
      driver: string;
      connection: string;
      sql: string;
    }) => api.ops(driver).query(connection, sql),
  });
}

export function useAnalyzeMutation() {
  return useMutation({
    mutationFn: ({
      driver,
      connection,
      table,
    }: {
      driver: string;
      connection: string;
      table: string;
    }) => api.ops(driver).analyze(connection, table),
  });
}
