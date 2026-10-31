import type { components } from "@/api/generated";
import { accessToken, authEnabled, clearSession } from "@/lib/auth";
import { PAGE_ROUTES } from "@/constants/routes";

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
export type SearchTarget = components["schemas"]["entity.SearchTarget"];
export type SearchRequest = components["schemas"]["entity.SearchRequest"];

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

// DriverInfo is one entry of the server's driver catalog: the driver name, the label a picker shows, and the console
// surfaces its module implements, so the console groups its navigation by what a driver can do.
export interface DriverInfo {
  driver: string;
  label: string;
  capabilities: string[];
}

// InstanceQuery filters the cross-driver instance list; empty fields are ignored.
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
  // kind is what the page claims; the server derives its own and refuses a disagreement
  kind?: string;
  scheduledAt?: number;
}

// ApiClient owns the transport: the base path, the auth header, and the 401 redirect live in one object.
export class ApiClient {
  constructor(private readonly basePath: string = "/external/v1") {}

  // path completes the injected base path for the endpoints that are not engine-scoped.
  private readonly path = (suffix: string) => `${this.basePath}${suffix}`;

  // endpoint builds an engine path from the driver, so no call hardcodes an engine URL segment: the server resolves a module by driver.
  private readonly endpoint = (driver: string, path: string) =>
    `${this.basePath}/engine/${encodeURIComponent(driver)}/${path}`;

  // connectionPath is the connection-scoped path; the name is URL-encoded so a space or slash cannot break the route.
  private readonly connectionPath = (driver: string, name: string, suffix = "") =>
    this.endpoint(driver, `connections/${encodeURIComponent(name)}${suffix}`);

  private readonly request = async <T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> => {
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
      window.location.assign(PAGE_ROUTES.LOGIN);
      throw new Error("redirecting to sign-in");
    }
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    if (!res.ok)
      throw new Error((data as { error?: string } | null)?.error ?? `HTTP ${res.status}`);
    return data as T;
  };

  // connectionOps is the connection lifecycle every driver shares; the driver decides the engine namespace, the routes are identical.
  private readonly connectionOps = (driver: string) => ({
    list: () =>
      this.request<{ connections: Connection[] }>("GET", this.endpoint(driver, "connections")),
    save: (c: Connection) =>
      this.request<Connection>("POST", this.endpoint(driver, "connections"), c),
    remove: (name: string) => this.request<null>("DELETE", this.connectionPath(driver, name)),
    test: (name: string) =>
      this.request<{ status: string }>("POST", this.connectionPath(driver, name, "/test")),
    // probe pings a draft before it exists, so the create page can verify and only then save
    probe: (c: Connection) =>
      this.request<{ status: string }>("POST", this.endpoint(driver, "connections/probe"), c),
  });

  // groups is the surface every page uses: one namespace per engine surface, each addressed by the driver that serves the connection.
  readonly groups = {
    me: () => this.request<Identity>("GET", this.path("/me")),
    // instances are the cross-driver view: one connection addressed by its stable id, whichever driver serves it.
    instances: {
      list: (query: InstanceQuery = {}) => {
        const params = new URLSearchParams();
        if (query.name) params.set("name", query.name);
        if (query.driver) params.set("driver", query.driver);
        if (query.host) params.set("host", query.host);
        if (query.page && query.page > 1) params.set("page", String(query.page));
        if (query.perPage) params.set("perPage", String(query.perPage));
        const qs = params.toString();
        return this.request<{ instances: Connection[]; total: number }>(
          "GET",
          this.path(`/instances${qs ? `?${qs}` : ""}`),
        );
      },
      get: (id: string) =>
        this.request<Connection>("GET", this.path(`/instances/${encodeURIComponent(id)}`)),
      // counts is one call for every driver, so the sidebar does not ask once per engine
      counts: () =>
        this.request<{ counts: Record<string, number> }>("GET", this.path("/instances/counts")),
      rotatePassword: (id: string, password: string) =>
        this.request<{ status: string }>(
          "POST",
          this.path(`/instances/${encodeURIComponent(id)}/password`),
          { password },
        ),
    },
    // connections is the SQL surface of one driver: the shared lifecycle plus schema, graph, info and rows.
    connections: (driver: string) => ({
      ...this.connectionOps(driver),
      schema: (name: string) =>
        this.request<{ tables: Table[] }>("GET", this.connectionPath(driver, name, "/schema")),
      erd: (name: string) =>
        this.request<SchemaGraph>("GET", this.connectionPath(driver, name, "/erd")),
      // info is what the server says about itself: the version it runs and what that version lets the console prove.
      info: (name: string) =>
        this.request<ServerInfo>("GET", this.connectionPath(driver, name, "/info")),
      rows: (name: string, schema: string, table: string, limit = 100) =>
        this.request<Result>(
          "GET",
          this.connectionPath(driver, name, `/tables/${table}/rows`) +
            `?schema=${encodeURIComponent(schema)}&limit=${limit}`,
        ),
    }),
    ops: (driver: string) => ({
      query: (name: string, sql: string) =>
        this.request<Result>("POST", this.connectionPath(driver, name, "/operations"), {
          op: "query",
          sql,
        }),
      rollback: (name: string, sql: string) =>
        this.request<Result>("POST", this.connectionPath(driver, name, "/operations"), {
          op: "rollback",
          sql,
        }),
      analyze: (name: string, table: string) =>
        this.request<Result>("POST", this.connectionPath(driver, name, "/operations"), {
          op: "analyze",
          table,
        }),
    }),
    memcache: (driver: string) => ({
      ...this.connectionOps(driver),
      keys: (name: string, pattern = "*", limit = 500) =>
        this.request<MemcacheKeyPage>(
          "GET",
          this.connectionPath(driver, name, "/keys") +
            `?pattern=${encodeURIComponent(pattern)}&limit=${limit}`,
        ),
      value: (name: string, key: string) =>
        this.request<Result>(
          "GET",
          this.connectionPath(driver, name, "/value") + `?key=${encodeURIComponent(key)}`,
        ),
      removeKeys: (name: string, keys: string[]) =>
        this.request<{ deleted: number }>("DELETE", this.connectionPath(driver, name, "/keys"), {
          keys,
        }),
      flush: (name: string) =>
        this.request<{ status: string }>("POST", this.connectionPath(driver, name, "/flush"), {
          confirm: name,
        }),
    }),
    search: (driver: string) => ({
      // what the platform indexes, and one query against a target: the query travels as the platform's own document
      targets: (name: string) =>
        this.request<{ targets: SearchTarget[] }>(
          "GET",
          this.connectionPath(driver, name, "/targets"),
        ),
      run: (name: string, request: SearchRequest) =>
        this.request<Result>("POST", this.connectionPath(driver, name, "/search"), request),
      changes: {
        // the search queue is its own: the same list endpoint, filtered to the kind this capability owns
        list: (connection?: string, status?: string) => {
          const params = new URLSearchParams({ kind: "search" });
          if (connection) params.set("connection", connection);
          if (status) params.set("status", status);
          return this.request<{ changes: Change[]; total: number }>(
            "GET",
            this.endpoint(driver, "changes") + `?${params.toString()}`,
          );
        },
        get: (id: string) => this.request<Change>("GET", this.endpoint(driver, `changes/${id}`)),
        approve: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/approve`)),
        apply: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/apply`)),
        reject: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/reject`)),
      },
    }),
    mongodb: (driver: string) => ({
      ...this.connectionOps(driver),
      databases: (name: string) =>
        this.request<{ databases: MongoDatabase[] }>(
          "GET",
          this.connectionPath(driver, name, "/databases"),
        ),
      collections: (name: string, database: string) =>
        this.request<{ collections: MongoCollection[] }>(
          "GET",
          this.connectionPath(driver, name, "/collections") +
            `?database=${encodeURIComponent(database)}`,
        ),
      // reads only: a write op is refused here and belongs in the request queue
      run: (name: string, req: MongoRequest) =>
        this.request<MongoResult>("POST", this.connectionPath(driver, name, "/requests"), req),
      changes: {
        // the request queue is the same list endpoint as the SQL queues: same filters, no kind
        list: (connection?: string, page?: number, status?: string, q?: string) => {
          const params = new URLSearchParams();
          if (connection) params.set("connection", connection);
          if (page && page > 1) params.set("page", String(page));
          if (status) params.set("status", status);
          if (q) params.set("q", q);
          const qs = params.toString();
          return this.request<{ changes: Change[]; total: number }>(
            "GET",
            this.endpoint(driver, "changes") + (qs ? `?${qs}` : ""),
          );
        },
        submit: (input: MongoChangeInput) =>
          this.request<Change>("POST", this.endpoint(driver, "changes"), input),
        get: (id: string) => this.request<Change>("GET", this.endpoint(driver, `changes/${id}`)),
        approve: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/approve`)),
        apply: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/apply`)),
        reject: (id: string) =>
          this.request<Change>("POST", this.endpoint(driver, `changes/${id}/reject`)),
      },
    }),
    changes: (driver: string) => ({
      // kind and q are the server's filters: they run before pagination, so the queue's total is its own count.
      list: (
        connection?: string,
        page?: number,
        perPage?: number,
        status?: string,
        kind?: string,
        q?: string,
      ) => {
        const params = new URLSearchParams();
        if (connection) params.set("connection", connection);
        if (page && page > 1) params.set("page", String(page));
        if (perPage) params.set("perPage", String(perPage));
        if (status) params.set("status", status);
        if (kind) params.set("kind", kind);
        if (q) params.set("q", q);
        const qs = params.toString();
        return this.request<{ changes: Change[]; total: number }>(
          "GET",
          this.endpoint(driver, "changes") + (qs ? `?${qs}` : ""),
        );
      },
      get: (id: string) => this.request<Change>("GET", this.endpoint(driver, `changes/${id}`)),
      submit: (input: ChangeInput) =>
        this.request<Change>("POST", this.endpoint(driver, "changes"), input),
      approve: (id: string) =>
        this.request<Change>("POST", this.endpoint(driver, `changes/${id}/approve`)),
      apply: (id: string) =>
        this.request<Change>("POST", this.endpoint(driver, `changes/${id}/apply`)),
      reject: (id: string) =>
        this.request<Change>("POST", this.endpoint(driver, `changes/${id}/reject`)),
      schedule: (id: string, scheduledAt: number) =>
        this.request<Change>("POST", this.endpoint(driver, `changes/${id}/schedule`), {
          scheduledAt,
        }),
    }),
    // config is the server's console-facing configuration (hardcoded Go-side). permissions: per-connection RBAC policy (users and groups), admin-managed.
    permissions: {
      // matrix is the per-instance overview: every instance with its whole policy.
      matrix: () => this.request<{ instances: InstanceAccess[] }>("GET", this.path("/access")),
      list: (driver: string, name: string) =>
        this.request<{ permissions: Permission[] }>(
          "GET",
          this.endpoint(driver, `connections/${encodeURIComponent(name)}/permissions`),
        ),
      set: (driver: string, name: string, member: string, memberType: string, level: string) =>
        this.request<Permission>(
          "PUT",
          this.endpoint(driver, `connections/${encodeURIComponent(name)}/permissions`),
          {
            member,
            memberType,
            level,
          },
        ),
      remove: (driver: string, name: string, member: string) =>
        this.request<null>(
          "DELETE",
          this.endpoint(
            driver,
            `connections/${encodeURIComponent(name)}/permissions?member=${encodeURIComponent(member)}`,
          ),
        ),
    },
    // accessRequests: a user asks for a level on a connection; an admin or that instance's admin decides.
    accessRequests: {
      create: (connection: string, level: string, reason: string) =>
        this.request<AccessRequest>("POST", this.path("/access-requests"), {
          connection,
          level,
          reason,
        }),
      mine: () =>
        this.request<{ requests: AccessRequest[] }>(
          "GET",
          this.path("/access-requests?scope=mine"),
        ),
      toApprove: () =>
        this.request<{ requests: AccessRequest[] }>(
          "GET",
          this.path("/access-requests?scope=to-approve"),
        ),
      approve: (id: string, note = "") =>
        this.request<AccessRequest>("POST", this.path(`/access-requests/${id}/approve`), { note }),
      reject: (id: string, note = "") =>
        this.request<AccessRequest>("POST", this.path(`/access-requests/${id}/reject`), { note }),
    },
    // users: the admin directory. Read-only plus the status/role controls; only the super admin may send a role.
    users: {
      list: (q: string, status: string) =>
        this.request<{ users: User[] }>(
          "GET",
          this.path(`/users?q=${encodeURIComponent(q)}&status=${encodeURIComponent(status)}`),
        ),
      get: (subject: string) =>
        this.request<UserDetail>("GET", this.path(`/users/${encodeURIComponent(subject)}`)),
      update: (subject: string, patch: { role?: UserRole }) =>
        this.request<User>("PATCH", this.path(`/users/${encodeURIComponent(subject)}`), patch),
    },
    config: {
      drivers: () => this.request<DriverInfo[]>("GET", this.path("/config/drivers")),
      levels: () => this.request<{ levels: AccessLevel[] }>("GET", this.path("/config/levels")),
      members: () =>
        this.request<{ groups: string[]; users: MemberUser[] }>(
          "GET",
          this.path("/config/members"),
        ),
    },
  };
}

// api is the console's one client; pages never construct their own.
export const api = new ApiClient().groups;

// ServerInfo is what a connected server reports about itself: the engine, the version it detected, and what that profile lets the console prove.
export interface ServerInfo {
  engine: string;
  version: string;
  capabilities: string[];
}
