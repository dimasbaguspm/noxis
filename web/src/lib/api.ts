import type { components } from "@/api/generated";

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
}

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

export const api = {
  connections: {
    list: () => request<{ connections: Connection[] }>("GET", "/v1/sql/connections"),
    save: (c: Connection) => request<Connection>("POST", "/v1/sql/connections", c),
    remove: (name: string) => request<null>("DELETE", `/v1/sql/connections/${name}`),
    test: (name: string) => request<{ status: string }>("POST", `/v1/sql/connections/${name}/test`),
    schema: (name: string) => request<{ tables: Table[] }>("GET", `/v1/sql/connections/${name}/schema`),
    stats: (name: string) => request<Stats>("GET", `/v1/sql/connections/${name}/stats`),
    statusStream: () => "/v1/connections/status/stream",
    // the live view subscribes to this endpoint with an EventSource
    statsStream: (name: string) => `/v1/sql/connections/${name}/stats/stream`,
    rows: (name: string, schema: string, table: string, limit = 100) =>
      request<Result>(
        "GET",
        `/v1/sql/connections/${name}/tables/${table}/rows?schema=${encodeURIComponent(schema)}&limit=${limit}`,
      ),
  },
  ops: {
    query: (name: string, sql: string) => request<Result>("POST", `/v1/sql/connections/${name}/query`, { sql }),
    alter: (name: string, sql: string) => request<Result>("POST", `/v1/sql/connections/${name}/alter`, { sql }),
    rollback: (name: string, sql: string) => request<Result>("POST", `/v1/sql/connections/${name}/rollback`, { sql }),
    analytics: (name: string, table: string) =>
      request<Result>("POST", `/v1/sql/connections/${name}/analytics`, { table }),
  },
  memcache: {
    list: () => request<{ connections: Connection[] }>("GET", "/v1/memcache/connections"),
    save: (c: Connection) => request<Connection>("POST", "/v1/memcache/connections", c),
    remove: (name: string) => request<null>("DELETE", `/v1/memcache/connections/${name}`),
    test: (name: string) => request<{ status: string }>("POST", `/v1/memcache/connections/${name}/test`),
    keys: (name: string, pattern = "*", limit = 500) =>
      request<MemcacheKeyPage>(
        "GET",
        `/v1/memcache/connections/${name}/keys?pattern=${encodeURIComponent(pattern)}&limit=${limit}`,
      ),
    value: (name: string, key: string) =>
      request<Result>("GET", `/v1/memcache/connections/${name}/value?key=${encodeURIComponent(key)}`),
    removeKeys: (name: string, keys: string[]) =>
      request<{ deleted: number }>("DELETE", `/v1/memcache/connections/${name}/keys`, { keys }),
    flush: (name: string) =>
      request<{ status: string }>("POST", `/v1/memcache/connections/${name}/flush`, { confirm: name }),
    stats: (name: string) => request<MemcacheStats>("GET", `/v1/memcache/connections/${name}/stats`),
    statsStream: (name: string) => `/v1/memcache/connections/${name}/stats/stream`,
  },
  mongodb: {
    list: () => request<{ connections: Connection[] }>("GET", "/v1/mongodb/connections"),
    save: (c: Connection) => request<Connection>("POST", "/v1/mongodb/connections", c),
    remove: (name: string) => request<null>("DELETE", `/v1/mongodb/connections/${name}`),
    test: (name: string) => request<{ status: string }>("POST", `/v1/mongodb/connections/${name}/test`),
    databases: (name: string) =>
      request<{ databases: MongoDatabase[] }>("GET", `/v1/mongodb/connections/${name}/databases`),
    collections: (name: string, database: string) =>
      request<{ collections: MongoCollection[] }>(
        "GET",
        `/v1/mongodb/connections/${name}/collections?database=${encodeURIComponent(database)}`,
      ),
    // reads only: a write op is refused here and belongs in the request queue
    run: (name: string, req: MongoRequest) =>
      request<MongoResult>("POST", `/v1/mongodb/connections/${name}/requests`, req),
    stats: (name: string, database?: string, collection?: string) =>
      request<MongoStats>(
        "GET",
        `/v1/mongodb/connections/${name}/stats?database=${encodeURIComponent(database ?? "")}` +
          `&collection=${encodeURIComponent(collection ?? "")}`,
      ),
    statsStream: (name: string, database?: string, collection?: string) =>
      `/v1/mongodb/connections/${name}/stats/stream?database=${encodeURIComponent(database ?? "")}` +
      `&collection=${encodeURIComponent(collection ?? "")}`,
    changes: {
      list: () => request<{ changes: Change[] }>("GET", "/v1/mongodb/changes"),
      submit: (input: MongoChangeInput) => request<Change>("POST", "/v1/mongodb/changes", input),
      get: (id: string) => request<Change>("GET", `/v1/mongodb/changes/${id}`),
      approve: (id: string) => request<Change>("POST", `/v1/mongodb/changes/${id}/approve`),
      apply: (id: string) => request<Change>("POST", `/v1/mongodb/changes/${id}/apply`),
      reject: (id: string) => request<Change>("POST", `/v1/mongodb/changes/${id}/reject`),
    },
  },
  changes: {
    list: () => request<{ changes: Change[] }>("GET", "/v1/sql/changes"),
    submit: (input: ChangeInput) => request<Change>("POST", "/v1/sql/changes", input),
    approve: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/approve`),
    apply: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/apply`),
    reject: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/reject`),
  },
};
