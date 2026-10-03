import type { components } from "@/api/generated";

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["entity.Result"];
export type Stats = components["schemas"]["sql.Stats"];
export type RedisStats = components["schemas"]["redis.Stats"];
export type RedisKey = components["schemas"]["redis.Key"];
export type RedisKeyPage = components["schemas"]["redis.KeyPage"];

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
  redis: {
    list: () => request<{ connections: Connection[] }>("GET", "/v1/redis/connections"),
    save: (c: Connection) => request<Connection>("POST", "/v1/redis/connections", c),
    remove: (name: string) => request<null>("DELETE", `/v1/redis/connections/${name}`),
    test: (name: string) => request<{ status: string }>("POST", `/v1/redis/connections/${name}/test`),
    keys: (name: string, pattern = "*", limit = 500) =>
      request<RedisKeyPage>(
        "GET",
        `/v1/redis/connections/${name}/keys?pattern=${encodeURIComponent(pattern)}&limit=${limit}`,
      ),
    value: (name: string, key: string) =>
      request<Result>("GET", `/v1/redis/connections/${name}/value?key=${encodeURIComponent(key)}`),
    removeKeys: (name: string, keys: string[]) =>
      request<{ deleted: number }>("DELETE", `/v1/redis/connections/${name}/keys`, { keys }),
    flush: (name: string) =>
      request<{ status: string }>("POST", `/v1/redis/connections/${name}/flush`, { confirm: name }),
    stats: (name: string) => request<RedisStats>("GET", `/v1/redis/connections/${name}/stats`),
    statsStream: (name: string) => `/v1/redis/connections/${name}/stats/stream`,
  },
  changes: {
    list: () => request<{ changes: Change[] }>("GET", "/v1/sql/changes"),
    submit: (input: ChangeInput) => request<Change>("POST", "/v1/sql/changes", input),
    approve: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/approve`),
    apply: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/apply`),
    reject: (id: string) => request<Change>("POST", `/v1/sql/changes/${id}/reject`),
  },
};
