// Thin typed client over the noxis API. Wire shapes come from doc/swagger.yaml
// via openapi-typescript (`pnpm gen:api`).
import type { components } from "./generated";

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["sql.Result"];

export interface Table {
  schema: string;
  name: string;
  columns: { name: string; type: string; nullable: boolean }[];
}

export interface ApiResult<T> {
  ok: boolean;
  status: number;
  data: T | null;
}

async function api<T>(method: string, path: string, body?: unknown): Promise<ApiResult<T>> {
  const res = await fetch(path, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  return { ok: res.ok, status: res.status, data: data as T | null };
}

export const listConnections = () => api<{ connections: Connection[] }>("GET", "/v1/connections");
export const saveConnection = (c: Connection) => api<Connection>("POST", "/v1/connections", c);
export const deleteConnection = (name: string) => api<null>("DELETE", `/v1/connections/${name}`);
export const testConnection = (name: string) => api<{ status: string }>("POST", `/v1/connections/${name}/test`);
export const getSchema = (name: string) => api<{ tables: Table[] }>("GET", `/v1/connections/${name}/schema`);
export const getRows = (name: string, schema: string, table: string, limit = 100) =>
  api<Result>("GET", `/v1/connections/${name}/tables/${table}/rows?schema=${encodeURIComponent(schema)}&limit=${limit}`);
export const runOp = (name: string, op: "query" | "alter" | "rollback" | "analytics", body: Record<string, string>) =>
  api<Result>("POST", `/v1/connections/${name}/${op}`, body);

export const listChanges = () => api<{ changes: Change[] }>("GET", "/v1/changes");
export const submitChange = (c: { connection: string; notes: string; query: string; rollback: string }) =>
  api<Change>("POST", "/v1/changes", c);
export const approveChange = (id: string) => api<Change>("POST", `/v1/changes/${id}/approve`);
export const applyChange = (id: string) => api<Change>("POST", `/v1/changes/${id}/apply`);
export const rejectChange = (id: string) => api<null>("DELETE", `/v1/changes/${id}`);

export const DRIVERS = ["postgres", "mysql"];
