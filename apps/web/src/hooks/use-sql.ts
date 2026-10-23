import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// useConnectionSchemaQuery returns the tables of one SQL connection; the
// completion and table browser read the same cached list.
export function useConnectionSchemaQuery(connection?: string) {
  return useQuery({
    queryKey: queryKeys.sql.schema(connection),
    queryFn: () => api.connections.schema(connection as string),
    enabled: !!connection,
    select: (res) => res.tables,
  });
}

// useErdQuery loads one connection's schema graph and its Mermaid diagram;
// enabled only while the diagram view is open, so browsing stays cheap.
export function useErdQuery(connection?: string, open = false) {
  return useQuery({
    queryKey: queryKeys.sql.erd(connection),
    queryFn: () => api.connections.erd(connection as string),
    enabled: open && !!connection,
  });
}

// useTableRowsQuery previews a table's rows; enabled only once a table is
// picked, so the cache re-runs when the selection changes.
export function useTableRowsQuery(connection?: string, schema?: string, table?: string) {
  return useQuery({
    queryKey: queryKeys.sql.rows(connection, schema, table),
    queryFn: () => api.connections.rows(connection as string, schema as string, table as string),
    enabled: !!connection && !!schema && !!table,
  });
}

// useRunQueryMutation executes a read against the instance; the result is the
// mutation's data, so a fresh run replaces the last result.
export function useRunQueryMutation() {
  return useMutation({
    mutationFn: ({ connection, sql }: { connection: string; sql: string }) => api.ops.query(connection, sql),
  });
}

export function useAnalyzeMutation() {
  return useMutation({
    mutationFn: ({ connection, table }: { connection: string; table: string }) =>
      api.ops.analyze(connection, table),
  });
}