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
    counts: pageKey("console.instances", "counts"),
  },
  // every engine read carries the driver, so two drivers' schemas, keys and databases can never share a cache entry
  sql: {
    schema: (driver: string | undefined, connection: string | undefined) =>
      pageKey("driver.query", "schema", driver, connection),
    erd: (driver: string | undefined, connection: string | undefined) =>
      pageKey("driver.erd", "erd", driver, connection),
    rows: (
      driver: string | undefined,
      connection: string | undefined,
      schema?: string,
      table?: string,
    ) => pageKey("driver.query", "rows", driver, connection, schema, table),
  },
  changes: {
    all: pageRoot("driver.changes.dml"),
    list: (
      driver: string | undefined,
      kind?: "dml" | "ddl",
      connections?: string,
      name?: string,
      status?: string,
      page = 1,
    ) => pageKey("driver.changes.dml", "list", driver, kind, connections, name, status, page),
    get: (driver: string | undefined, id: string) =>
      pageKey("driver.changes.dml", "get", driver, id),
  },
  keys: {
    all: pageRoot("driver.keys"),
    keys: (
      driver: string | undefined,
      connection: string | undefined,
      pattern?: string,
      page = 1,
    ) => pageKey("driver.keys", "keys", driver, connection, pattern, page),
    value: (driver: string | undefined, connection: string | undefined, key?: string) =>
      pageKey("driver.keys", "value", driver, connection, key),
  },
  search: {
    all: pageRoot("driver.search"),
    targets: (driver: string | undefined, connection: string | undefined) =>
      pageKey("driver.search", "targets", driver, connection),
  },
  explore: {
    databases: (driver: string | undefined, connection: string | undefined) =>
      pageKey("driver.documents", "databases", driver, connection),
    collections: (driver: string | undefined, connection: string | undefined, database?: string) =>
      pageKey("driver.documents", "collections", driver, connection, database),
  },
  requests: {
    all: pageRoot("driver.requests"),
    list: (
      driver: string | undefined,
      connections?: string,
      name?: string,
      status?: string,
      page = 1,
    ) => pageKey("driver.requests", "list", driver, connections, name, status, page),
    get: (driver: string | undefined, id: string) =>
      pageKey("driver.requests.detail", "get", driver, id),
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
