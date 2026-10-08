import type { components } from "@/api/generated";
import { FAMILY, type FamilyKey } from "@/lib/families";

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["entity.Result"];
export type Stats = components["schemas"]["sql.Stats"];
export type MemcacheStats = components["schemas"]["memcache.Stats"];
export type MongoStats = components["schemas"]["mongodb.Stats"];
export type MongoDatabase = components["schemas"]["mongodb.Database"];
export type MongoCollection = components["schemas"]["mongodb.Collection"];
export type MongoRequest = components["schemas"]["mongodb.Request"];
export type MongoResult = components["schemas"]["mongodb.Result"];

// MongoChangeInput is one submitted request: the change envelope plus the
// request itself, flat so the wizard sends a single object.
export interface MongoChangeInput extends MongoRequest {
  connection: string;
  notes: string;
}
export type MemcacheKey = components["schemas"]["memcache.Key"];
export type MemcacheKeyPage = components["schemas"]["memcache.KeyPage"];

export interface Column {
  name: string;
  type: string;
  nullable: boolean;
}
export interface Table {
  schema: string;
  name: string;
  columns: Column[];
}
export interface ChangeInput {
  connection: string;
  notes: string;
  query: string;
  rollback: string;
  scheduledAt?: number;
}

// v1 builds an endpoint from the engine family, so no call hardcodes a family
// URL prefix: the family is the server's generic route parameter.
const v1 = (family: FamilyKey, path: string) => `/v1/${family}/${path}`;

// connPath is the connection-scoped path; the name is URL-encoded so a name
// with a space or slash cannot break the route.
const connPath = (family: FamilyKey, name: string, suffix = "") =>
  v1(family, `connections/${encodeURIComponent(name)}${suffix}`);

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error((data as { error?: string } | null)?.error ?? `HTTP ${res.status}`);
  return data as T;
}

// connectionOps is the connection lifecycle every family shares; the family
// decides the namespace, the routes are identical.
function connectionOps(family: FamilyKey) {
  return {
    list: () => request<{ connections: Connection[] }>("GET", v1(family, "connections")),
    save: (c: Connection) => request<Connection>("POST", v1(family, "connections"), c),
    remove: (name: string) => request<null>("DELETE", connPath(family, name)),
    test: (name: string) => request<{ status: string }>("POST", connPath(family, name, "/test")),
  };
}

export const api = {
  connections: {
    ...connectionOps(FAMILY.SQL),
    schema: (name: string) => request<{ tables: Table[] }>("GET", connPath(FAMILY.SQL, name, "/schema")),
    stats: (name: string) => request<Stats>("GET", connPath(FAMILY.SQL, name, "/stats")),
    // the status stream probes every family, so it is not family-scoped
    statusStream: () => "/v1/connections/status/stream",
    // the live view subscribes to this endpoint with an EventSource
    statsStream: (name: string) => connPath(FAMILY.SQL, name, "/stats/stream"),
    rows: (name: string, schema: string, table: string, limit = 100) =>
      request<Result>(
        "GET",
        connPath(FAMILY.SQL, name, `/tables/${table}/rows`) +
          `?schema=${encodeURIComponent(schema)}&limit=${limit}`,
      ),
  },
  ops: {
    query: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/query"), { sql }),
    alter: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/alter"), { sql }),
    rollback: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/rollback"), { sql }),
    analytics: (name: string, table: string) =>
      request<Result>("POST", connPath(FAMILY.SQL, name, "/analytics"), { table }),
  },
  memcache: {
    ...connectionOps(FAMILY.MEMCACHE),
    keys: (name: string, pattern = "*", limit = 500) =>
      request<MemcacheKeyPage>(
        "GET",
        connPath(FAMILY.MEMCACHE, name, "/keys") +
          `?pattern=${encodeURIComponent(pattern)}&limit=${limit}`,
      ),
    value: (name: string, key: string) =>
      request<Result>("GET", connPath(FAMILY.MEMCACHE, name, "/value") + `?key=${encodeURIComponent(key)}`),
    removeKeys: (name: string, keys: string[]) =>
      request<{ deleted: number }>("DELETE", connPath(FAMILY.MEMCACHE, name, "/keys"), { keys }),
    flush: (name: string) =>
      request<{ status: string }>("POST", connPath(FAMILY.MEMCACHE, name, "/flush"), { confirm: name }),
    stats: (name: string) => request<MemcacheStats>("GET", connPath(FAMILY.MEMCACHE, name, "/stats")),
    statsStream: (name: string) => connPath(FAMILY.MEMCACHE, name, "/stats/stream"),
  },
  mongodb: {
    ...connectionOps(FAMILY.MONGODB),
    databases: (name: string) =>
      request<{ databases: MongoDatabase[] }>("GET", connPath(FAMILY.MONGODB, name, "/databases")),
    collections: (name: string, database: string) =>
      request<{ collections: MongoCollection[] }>(
        "GET",
        connPath(FAMILY.MONGODB, name, "/collections") + `?database=${encodeURIComponent(database)}`,
      ),
    // reads only: a write op is refused here and belongs in the request queue
    run: (name: string, req: MongoRequest) =>
      request<MongoResult>("POST", connPath(FAMILY.MONGODB, name, "/requests"), req),
    stats: (name: string, database?: string, collection?: string) =>
      request<MongoStats>(
        "GET",
        connPath(FAMILY.MONGODB, name, "/stats") +
          `?database=${encodeURIComponent(database ?? "")}&collection=${encodeURIComponent(collection ?? "")}`,
      ),
    statsStream: (name: string, database?: string, collection?: string) =>
      connPath(FAMILY.MONGODB, name, "/stats/stream") +
      `?database=${encodeURIComponent(database ?? "")}&collection=${encodeURIComponent(collection ?? "")}`,
    changes: {
      list: () => request<{ changes: Change[] }>("GET", v1(FAMILY.MONGODB, "changes")),
      submit: (input: MongoChangeInput) => request<Change>("POST", v1(FAMILY.MONGODB, "changes"), input),
      get: (id: string) => request<Change>("GET", v1(FAMILY.MONGODB, `changes/${id}`)),
      approve: (id: string) => request<Change>("POST", v1(FAMILY.MONGODB, `changes/${id}/approve`)),
      apply: (id: string) => request<Change>("POST", v1(FAMILY.MONGODB, `changes/${id}/apply`)),
      reject: (id: string) => request<Change>("POST", v1(FAMILY.MONGODB, `changes/${id}/reject`)),
    },
  },
  changes: {
    list: () => request<{ changes: Change[] }>("GET", v1(FAMILY.SQL, "changes")),
    submit: (input: ChangeInput) => request<Change>("POST", v1(FAMILY.SQL, "changes"), input),
    approve: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/approve`)),
    apply: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/apply`)),
    reject: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/reject`)),
  },
  // family returns the shared connection lifecycle for any engine family.
  family: connectionOps,
};
