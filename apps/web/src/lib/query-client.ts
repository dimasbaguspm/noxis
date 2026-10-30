import { QueryClient } from "@tanstack/react-query";
import { ROUTES, type RouteId } from "@/constants/routes";

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

// pageKey starts a cache key at the section and entity of the page that owns the data,
// so a key cannot drift from the route it belongs to.
const pageKey = (route: RouteId, ...rest: unknown[]) =>
  [ROUTES[route].section, ROUTES[route].entity, ...rest] as const;

// pageRoot is the prefix that invalidates every entry a page owns.
const pageRoot = (route: RouteId) => [ROUTES[route].section, ROUTES[route].entity] as const;

// queryKeys centralizes every cache key so invalidations never drift from the reads.
export const queryKeys = {
  // the identity is the account section's own record, so it lives under its section like any other read
  me: pageRoot("my.profile"),
  // auth has no page either: the session bootstrap runs before any section renders
  auth: {
    session: ["auth", "session"] as const,
  },
  // config has no page of its own: the driver catalog, the level list, and the member list feed forms in several sections
  config: {
    drivers: ["config", "drivers"] as const,
    levels: ["config", "levels"] as const,
    members: ["config", "members"] as const,
  },
  instances: {
    all: pageRoot("console.instances"),
    list: (query?: { name?: string; driver?: string; host?: string; page?: number }) =>
      pageKey("console.instances", "list", query),
    get: (id: string) => pageKey("console.instances.detail", "get", id),
    schema: (id: string) => pageKey("console.instances.detail", "schema", id),
  },
  sql: {
    schema: (connection: string | undefined) => pageKey("console.sql.query", "schema", connection),
    erd: (connection: string | undefined) => pageKey("console.sql.erd", "erd", connection),
    rows: (connection: string | undefined, schema?: string, table?: string) =>
      pageKey("console.sql.query", "rows", connection, schema, table),
    changes: {
      all: pageRoot("console.sql.dml"),
      list: (kind?: "dml" | "ddl", connection?: string, status?: string, page = 1) =>
        pageKey("console.sql.dml", "list", kind, connection, status, page),
      get: (id: string) => pageKey("console.sql.dml", "get", id),
    },
  },
  memcache: {
    all: pageRoot("console.memcache"),
    keys: (connection: string | undefined, pattern?: string, page = 1) =>
      pageKey("console.memcache", "keys", connection, pattern, page),
    value: (connection: string | undefined, key?: string) =>
      pageKey("console.memcache", "value", connection, key),
  },
  mongodb: {
    databases: (connection: string | undefined) =>
      pageKey("console.mongodb.explore", "databases", connection),
    collections: (connection: string | undefined, database?: string) =>
      pageKey("console.mongodb.explore", "collections", connection, database),
    requests: {
      all: pageRoot("console.mongodb.requests"),
      list: () => pageKey("console.mongodb.requests", "list"),
      get: (id: string) => pageKey("console.mongodb.requests.detail", "get", id),
    },
  },
  access: {
    // per-instance grants belong to the instance access tab; the approval inbox belongs to the admin section
    all: pageRoot("console.instances.access"),
    permissions: (driver: string | undefined, name: string) =>
      pageKey("console.instances.access", "permissions", driver, name),
    matrix: pageKey("admin.access", "matrix"),
    requests: {
      all: pageRoot("admin.access"),
      mine: pageKey("admin.access", "requests", "mine"),
      toApprove: pageKey("admin.access", "requests", "to-approve"),
    },
  },
  users: {
    all: pageRoot("admin.users"),
    list: (q: string, status: string) => pageKey("admin.users", "list", q, status),
    get: (subject: string) => pageKey("admin.users.detail", "get", subject),
  },
};
