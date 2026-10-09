import type { components } from "@/api/generated";
import { FAMILY, type FamilyKey } from "@/lib/families";
import { accessToken, authEnabled, signin } from "@/lib/auth";

export type Level = "read" | "write" | "admin";
export type Identity = components["schemas"]["controller.MeResponse"];

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["entity.Result"];
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
export interface BatchFileInput {
  name: string;
  query: string;
  rollback: string;
}
export interface BatchInput {
  connection: string;
  notes: string;
  files: BatchFileInput[];
}
export interface BatchFileResult {
  name: string;
  ok: boolean;
  error?: string;
  change?: Change;
}
export interface BatchResult {
  batchId: string;
  results: BatchFileResult[];
}

// v1 builds an endpoint from the engine family, so no call hardcodes a family
// URL prefix: the family is the server's generic route parameter.
const v1 = (family: FamilyKey, path: string) => `/v1/engine/${family}/${path}`;

// connPath is the connection-scoped path; the name is URL-encoded so a name
// with a space or slash cannot break the route.
const connPath = (family: FamilyKey, name: string, suffix = "") =>
  v1(family, `connections/${encodeURIComponent(name)}${suffix}`);

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const token = await accessToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(path, {
    method,
    headers: Object.keys(headers).length > 0 ? headers : undefined,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  // an expired or revoked token drops the console back to the sign-in page
  if (res.status === 401 && authEnabled()) {
    await signin();
    throw new Error("redirecting to sign-in");
  }
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
  me: () => request<Identity>("GET", "/v1/me"),
  // instances are the cross-family view: one connection addressed by its
  // stable id, whatever family serves it.
  instances: {
    get: (id: string) => request<Connection>("GET", `/v1/instances/${encodeURIComponent(id)}`),
    changes: (id: string) =>
      request<{ changes: Change[] }>("GET", `/v1/instances/${encodeURIComponent(id)}/changes`),
  },
  connections: {
    ...connectionOps(FAMILY.SQL),
    schema: (name: string) => request<{ tables: Table[] }>("GET", connPath(FAMILY.SQL, name, "/schema")),
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
    analyze: (name: string, table: string) =>
      request<Result>("POST", connPath(FAMILY.SQL, name, "/analyze"), { table }),
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
    submitBatch: (input: BatchInput) => request<BatchResult>("POST", v1(FAMILY.SQL, "changes/batch"), input),
    approve: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/approve`)),
    apply: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/apply`)),
    reject: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/reject`)),
  },
  // family returns the shared connection lifecycle for any engine family.
  family: connectionOps,
};
