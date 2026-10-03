// Thin typed client over the master API. Wire shapes come from
// doc/swagger.yaml via openapi-typescript (`pnpm gen:api`).
import type { components } from "./generated";

export type Engine = NonNullable<components["schemas"]["entity.Spec"]["engines"]>[number];
export type NodeStatus = components["schemas"]["entity.NodeStatus"];
export type OpResult = components["schemas"]["control.OpRequest"] & { output?: string; error?: string };

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

export const getStatus = () => api<{ nodes: NodeStatus[] }>("GET", "/v1/status");
export const getSpec = (node: string) => api<{ node: string; engines: Engine[] }>("GET", `/v1/nodes/${node}/spec`);
export const putSpec = (node: string, engines: Engine[]) =>
  api<null>("PUT", `/v1/nodes/${node}/spec`, { engines });
export const runOp = (node: string, op: string, body: Record<string, string>) =>
  api<OpResult>("POST", `/v1/nodes/${node}/doorman/${op}`, body);

export const ENGINE_TYPES: Record<string, string[]> = {
  postgres: ["16-alpine", "16.6-alpine", "17-alpine"],
  mariadb: ["11"],
  mysql: ["8"],
};