import type { components } from "@/api/generated";

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["sql.Result"];

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
    list: () => request<{ connections: Connection[] }>("GET", "/v1/connections"),
    save: (c: Connection) => request<Connection>("POST", "/v1/connections", c),
    remove: (name: string) => request<null>("DELETE", `/v1/connections/${name}`),
    test: (name: string) => request<{ status: string }>("POST", `/v1/connections/${name}/test`),
    schema: (name: string) => request<{ tables: Table[] }>("GET", `/v1/connections/${name}/schema`),
    rows: (name: string, schema: string, table: string, limit = 100) =>
      request<Result>(
        "GET",
        `/v1/connections/${name}/tables/${table}/rows?schema=${encodeURIComponent(schema)}&limit=${limit}`,
      ),
  },
  ops: {
    query: (name: string, sql: string) => request<Result>("POST", `/v1/connections/${name}/query`, { sql }),
    alter: (name: string, sql: string) => request<Result>("POST", `/v1/connections/${name}/alter`, { sql }),
    rollback: (name: string, sql: string) => request<Result>("POST", `/v1/connections/${name}/rollback`, { sql }),
    analytics: (name: string, table: string) =>
      request<Result>("POST", `/v1/connections/${name}/analytics`, { table }),
  },
  changes: {
    list: () => request<{ changes: Change[] }>("GET", "/v1/changes"),
    submit: (input: ChangeInput) => request<Change>("POST", "/v1/changes", input),
    approve: (id: string) => request<Change>("POST", `/v1/changes/${id}/approve`),
    apply: (id: string) => request<Change>("POST", `/v1/changes/${id}/apply`),
    reject: (id: string) => request<null>("DELETE", `/v1/changes/${id}`),
  },
};
