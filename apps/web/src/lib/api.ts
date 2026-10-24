import type { components } from "@/api/generated";
import { FAMILY, type FamilyKey } from "@/lib/families";
import { accessToken, authEnabled, clearSession } from "@/lib/auth";

export type Level = "read" | "write" | "admin";
export type Identity = components["schemas"]["entity.MeResponse"];

export type Connection = components["schemas"]["entity.Connection"];
export type Change = components["schemas"]["entity.Change"];
export type ChangeCheck = components["schemas"]["entity.ChangeCheck"];
export type Result = components["schemas"]["entity.Result"];
export type MongoDatabase = components["schemas"]["entity.Database"];
export type MongoCollection = components["schemas"]["entity.Collection"];
export type MongoRequest = components["schemas"]["entity.MongoRequest"];
export type MongoResult = components["schemas"]["entity.Result"];

// MongoChangeInput is one submitted request: the change envelope plus the request itself, flat so the wizard sends a single object.
export interface MongoChangeInput extends MongoRequest {
  connection: string;
  notes: string;
}
export type MemcacheKey = components["schemas"]["entity.Key"];
export type MemcacheKeyPage = components["schemas"]["entity.KeyPage"];
export type Permission = components["schemas"]["entity.Permission"];
export type MemberUser = components["schemas"]["entity.Member"];
export type User = components["schemas"]["entity.User"];
export type UserRole = User["role"];
export type UserStatus = User["status"];
export type InstanceAccess = components["schemas"]["entity.InstanceAccess"];
export type SchemaGraph = components["schemas"]["entity.SchemaGraph"];
export type UserDetail = components["schemas"]["entity.UserDetail"];
export type AccessEvent = components["schemas"]["entity.AccessEvent"];
export type AccessLevel = components["schemas"]["entity.LevelInfo"];
export type AccessRequest = components["schemas"]["entity.AccessRequest"];

// DriverInfo is one entry of the server's hardcoded driver catalog.
export interface DriverInfo {
  driver: string;
  family: FamilyKey;
  label: string;
}

// InstanceQuery filters the cross-family instance list; empty fields are ignored.
export interface InstanceQuery {
  name?: string;
  driver?: string;
  host?: string;
  page?: number;
  perPage?: number;
}

export interface Column {
  name: string;
  type: string;
  nullable: boolean;
}
export interface Table {
  id?: string;
  schema: string;
  name: string;
  comment?: string;
  columns: Column[];
  indexes?: { name: string; columns: string[]; unique?: boolean; primary?: boolean }[];
  estimatedRows?: number;
  sizeBytes?: number;
}
export interface ChangeInput {
  connection: string;
  notes: string;
  query: string;
  rollback: string;
  scheduledAt?: number;
}

// v1 builds an endpoint from the engine family, so no call hardcodes a family URL prefix: the family is the server's generic route parameter.
const v1 = (family: FamilyKey, path: string) => `/external/v1/engine/${family}/${path}`;

// connPath is the connection-scoped path; the name is URL-encoded so a name with a space or slash cannot break the route.
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
  // an expired or revoked token drops the console to the login page; the user restarts the OIDC flow from there.
  if (res.status === 401 && authEnabled()) {
    await clearSession();
    window.location.assign("/login");
    throw new Error("redirecting to sign-in");
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error((data as { error?: string } | null)?.error ?? `HTTP ${res.status}`);
  return data as T;
}

// connectionOps is the connection lifecycle every family shares; the family decides the namespace, the routes are identical.
function connectionOps(family: FamilyKey) {
  return {
    list: () => request<{ connections: Connection[] }>("GET", v1(family, "connections")),
    save: (c: Connection) => request<Connection>("POST", v1(family, "connections"), c),
    remove: (name: string) => request<null>("DELETE", connPath(family, name)),
    test: (name: string) => request<{ status: string }>("POST", connPath(family, name, "/test")),
  };
}

export const api = {
  me: () => request<Identity>("GET", "/external/v1/me"),
  // instances are the cross-family view: one connection addressed by its stable id, whatever family serves it.
  instances: {
    list: (query: InstanceQuery = {}) => {
      const params = new URLSearchParams();
      if (query.name) params.set("name", query.name);
      if (query.driver) params.set("driver", query.driver);
      if (query.host) params.set("host", query.host);
      if (query.page && query.page > 1) params.set("page", String(query.page));
      if (query.perPage) params.set("perPage", String(query.perPage));
      const qs = params.toString();
      return request<{ instances: Connection[]; total: number }>("GET", `/external/v1/instances${qs ? `?${qs}` : ""}`);
    },
    get: (id: string) => request<Connection>("GET", `/external/v1/instances/${encodeURIComponent(id)}`),
    rotatePassword: (id: string, password: string) =>
      request<{ status: string }>("POST", `/external/v1/instances/${encodeURIComponent(id)}/password`, { password }),
  },
  connections: {
    ...connectionOps(FAMILY.SQL),
    schema: (name: string) => request<{ tables: Table[] }>("GET", connPath(FAMILY.SQL, name, "/schema")),
    erd: (name: string) => request<SchemaGraph>("GET", connPath(FAMILY.SQL, name, "/erd")),
    rows: (name: string, schema: string, table: string, limit = 100) =>
      request<Result>(
        "GET",
        connPath(FAMILY.SQL, name, `/tables/${table}/rows`) +
          `?schema=${encodeURIComponent(schema)}&limit=${limit}`,
      ),
  },
  ops: {
    query: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/operations"), { op: "query", sql }),
    alter: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/operations"), { op: "alter", sql }),
    rollback: (name: string, sql: string) => request<Result>("POST", connPath(FAMILY.SQL, name, "/operations"), { op: "rollback", sql }),
    analyze: (name: string, table: string) =>
      request<Result>("POST", connPath(FAMILY.SQL, name, "/operations"), { op: "analyze", table }),
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
    list: (connection?: string, page?: number, perPage?: number, status?: string) => {
      const params = new URLSearchParams();
      if (connection) params.set("connection", connection);
      if (page && page > 1) params.set("page", String(page));
      if (perPage) params.set("perPage", String(perPage));
      if (status) params.set("status", status);
      const qs = params.toString();
      return request<{ changes: Change[]; total: number }>(
        "GET",
        v1(FAMILY.SQL, "changes") + (qs ? `?${qs}` : ""),
      );
    },
    get: (id: string) => request<Change>("GET", v1(FAMILY.SQL, `changes/${id}`)),
    submit: (input: ChangeInput) => request<Change>("POST", v1(FAMILY.SQL, "changes"), input),
    approve: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/approve`)),
    apply: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/apply`)),
    reject: (id: string) => request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/reject`)),
    schedule: (id: string, scheduledAt: number) =>
      request<Change>("POST", v1(FAMILY.SQL, `changes/${id}/schedule`), { scheduledAt }),
  },
  // config is the server's console-facing configuration (hardcoded Go-side). permissions: per-connection RBAC policy (users and groups), admin-managed.
  permissions: {
    // matrix is the per-instance overview: every instance with its whole policy.
    matrix: () => request<{ instances: InstanceAccess[] }>("GET", "/external/v1/access"),
    list: (family: FamilyKey, name: string) =>
      request<{ permissions: Permission[] }>("GET", v1(family, `connections/${encodeURIComponent(name)}/permissions`)),
    set: (family: FamilyKey, name: string, member: string, memberType: string, level: string) =>
      request<Permission>("PUT", v1(family, `connections/${encodeURIComponent(name)}/permissions`), {
        member,
        memberType,
        level,
      }),
    remove: (family: FamilyKey, name: string, member: string) =>
      request<null>(
        "DELETE",
        v1(family, `connections/${encodeURIComponent(name)}/permissions?member=${encodeURIComponent(member)}`),
      ),
  },
  // accessRequests: a user asks for a level on a connection; an admin or that instance's admin decides.
  accessRequests: {
    create: (connection: string, level: string, reason: string) =>
      request<AccessRequest>("POST", "/external/v1/access-requests", { connection, level, reason }),
    mine: () => request<{ requests: AccessRequest[] }>("GET", "/external/v1/access-requests?scope=mine"),
    toApprove: () => request<{ requests: AccessRequest[] }>("GET", "/external/v1/access-requests?scope=to-approve"),
    approve: (id: string, note = "") =>
      request<AccessRequest>("POST", `/external/v1/access-requests/${id}/approve`, { note }),
    reject: (id: string, note = "") =>
      request<AccessRequest>("POST", `/external/v1/access-requests/${id}/reject`, { note }),
  },
  // users: the admin directory. Read-only plus the status/role controls; only the super admin may send a role.
  users: {
    list: (q: string, status: string) =>
      request<{ users: User[] }>(
        "GET",
        `/external/v1/users?q=${encodeURIComponent(q)}&status=${encodeURIComponent(status)}`,
      ),
    get: (subject: string) => request<UserDetail>("GET", `/external/v1/users/${encodeURIComponent(subject)}`),
    update: (subject: string, patch: { role?: UserRole }) =>
      request<User>("PATCH", `/external/v1/users/${encodeURIComponent(subject)}`, patch),
  },
  config: {
    drivers: () => request<DriverInfo[]>("GET", "/external/v1/config/drivers"),
    levels: () => request<{ levels: AccessLevel[] }>("GET", "/external/v1/config/levels"),
    members: () => request<{ groups: string[]; users: MemberUser[] }>("GET", "/external/v1/config/members"),
  },
  // family returns the shared connection lifecycle for any engine family.
  family: connectionOps,
};
