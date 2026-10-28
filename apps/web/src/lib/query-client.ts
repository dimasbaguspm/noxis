import { QueryClient } from "@tanstack/react-query";

// The shared QueryClient centralizes keys and invalidation; window refocus does not refetch.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

// queryKeys centralizes every cache key so invalidations never drift from the reads.
export const queryKeys = {
  me: ["me"] as const,
  config: {
    drivers: ["config", "drivers"] as const,
  },
  instances: {
    list: (query?: { name?: string; driver?: string; host?: string; page?: number }) =>
      ["instances", "list", query] as const,
    get: (id: string) => ["instances", "get", id] as const,
    schema: (id: string) => ["instances", "schema", id] as const,
  },
  sql: {
    schema: (connection: string | undefined) => ["sql", "schema", connection] as const,
    erd: (connection: string | undefined) => ["sql", "erd", connection] as const,
    rows: (connection: string | undefined, schema?: string, table?: string) =>
      ["sql", "rows", connection, schema, table] as const,
    changes: {
      list: (kind?: "dml" | "ddl", connection?: string, status?: string, page = 1) =>
        ["sql", "changes", "list", kind, connection, status, page] as const,
      get: (id: string) => ["sql", "changes", "get", id] as const,
    },
  },
  memcache: {
    keys: (connection: string | undefined, pattern?: string, page = 1) =>
      ["memcache", "keys", connection, pattern, page] as const,
    key: (connection: string | undefined, key?: string) =>
      ["memcache", "value", connection, key] as const,
  },
  mongodb: {
    databases: (connection: string | undefined) => ["mongodb", "databases", connection] as const,
    collections: (connection: string | undefined, database?: string) =>
      ["mongodb", "collections", connection, database] as const,
    requests: {
      list: () => ["mongodb", "requests", "list"] as const,
      get: (id: string) => ["mongodb", "requests", "get", id] as const,
    },
  },
};
