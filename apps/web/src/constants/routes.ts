// Route metadata is the single source for the router, the sidebar, the breadcrumb, the document title, and the cache keys.

export type RouteSection = "console" | "my" | "admin" | "auth";

export interface RouteMeta {
  // path is the pattern the router matches; ":name" segments are parameters.
  path: string;
  // title is the document title and the page heading.
  title: string;
  // label is the sidebar entry and the breadcrumb leaf.
  label: string;
  // entity is the API entity, and the first segment of the page's cache key.
  entity: string;
  section: RouteSection;
}

// CONSOLE is the base path of every database engine page, so the URL says which
// product section the user is in. Each future section (object storage,
// containers, config) gets a base of its own next to it.
export const CONSOLE = "/console";

export const ROUTES = {
  "console.instances": {
    path: `${CONSOLE}/instances`,
    title: "Instances",
    label: "Instances",
    entity: "instances",
    section: "console",
  },
  "console.instances.detail": {
    path: `${CONSOLE}/instances/:id`,
    title: "Instance",
    label: "Instance",
    entity: "instances",
    section: "console",
  },
  "console.instances.tables": {
    path: `${CONSOLE}/instances/:id/tables`,
    title: "Tables",
    label: "Tables",
    entity: "instances",
    section: "console",
  },
  "console.instances.erd": {
    path: `${CONSOLE}/instances/:id/erd`,
    title: "ERD",
    label: "ERD",
    entity: "instances",
    section: "console",
  },
  "console.instances.documents": {
    path: `${CONSOLE}/instances/:id/documents`,
    title: "Documents",
    label: "Documents",
    entity: "instances",
    section: "console",
  },
  "console.instances.keys": {
    path: `${CONSOLE}/instances/:id/keys`,
    title: "Keys",
    label: "Keys",
    entity: "instances",
    section: "console",
  },
  "console.instances.access": {
    path: `${CONSOLE}/instances/:id/access`,
    title: "Access",
    label: "Access",
    entity: "access",
    section: "console",
  },
  "console.sql.query": {
    path: `${CONSOLE}/sql/query`,
    title: "Query",
    label: "Query",
    entity: "sql",
    section: "console",
  },
  "console.sql.erd": {
    path: `${CONSOLE}/sql/erd`,
    title: "ERD",
    label: "ERD",
    entity: "sql",
    section: "console",
  },
  "console.sql.dml": {
    path: `${CONSOLE}/sql/dml`,
    title: "DML changes",
    label: "DML",
    entity: "changes",
    section: "console",
  },
  "console.sql.dml.detail": {
    path: `${CONSOLE}/sql/dml/:id`,
    title: "DML change",
    label: "DML change",
    entity: "changes",
    section: "console",
  },
  "console.sql.ddl": {
    path: `${CONSOLE}/sql/ddl`,
    title: "DDL changes",
    label: "DDL",
    entity: "changes",
    section: "console",
  },
  "console.sql.ddl.detail": {
    path: `${CONSOLE}/sql/ddl/:id`,
    title: "DDL change",
    label: "DDL change",
    entity: "changes",
    section: "console",
  },
  "console.memcache": {
    path: `${CONSOLE}/memcache`,
    title: "Memory Cache",
    label: "Keys",
    entity: "memcache",
    section: "console",
  },
  "console.mongodb.explore": {
    path: `${CONSOLE}/mongodb/explore`,
    title: "Explore",
    label: "Explore",
    entity: "mongodb",
    section: "console",
  },
  "console.mongodb.requests": {
    path: `${CONSOLE}/mongodb/requests`,
    title: "Requests",
    label: "Requests",
    entity: "requests",
    section: "console",
  },
  "console.mongodb.requests.new": {
    path: `${CONSOLE}/mongodb/requests/new`,
    title: "New request",
    label: "New request",
    entity: "requests",
    section: "console",
  },
  "console.mongodb.requests.detail": {
    path: `${CONSOLE}/mongodb/requests/:id`,
    title: "Request",
    label: "Request",
    entity: "requests",
    section: "console",
  },
  "my.profile": {
    path: "/my/profile",
    title: "Profile",
    label: "Profile",
    entity: "me",
    section: "my",
  },
  "my.settings": {
    path: "/my/settings",
    title: "Settings",
    label: "Settings",
    entity: "me",
    section: "my",
  },
  "my.access": {
    path: "/my/access",
    title: "Access",
    label: "Access",
    entity: "access",
    section: "my",
  },
  "admin.users": {
    path: "/users",
    title: "Users",
    label: "Users",
    entity: "users",
    section: "admin",
  },
  "admin.users.detail": {
    path: "/users/:subject",
    title: "Account",
    label: "Account",
    entity: "users",
    section: "admin",
  },
  "admin.access": {
    path: "/access",
    title: "Access",
    label: "Access",
    entity: "access",
    section: "admin",
  },
  "auth.login": {
    path: "/login",
    title: "Sign in",
    label: "Sign in",
    entity: "auth",
    section: "auth",
  },
  "auth.callback": {
    path: "/auth/callback",
    title: "Signing in",
    label: "Signing in",
    entity: "auth",
    section: "auth",
  },
} as const satisfies Record<string, RouteMeta>;

export type RouteId = keyof typeof ROUTES;

// PREFIX_LABELS words the URL segments that group routes but are not pages themselves.
const PREFIX_LABELS: Record<string, string> = {
  console: "Console",
  instances: "Instances",
  sql: "SQL",
  memcache: "Memory Cache",
  mongodb: "MongoDB",
  my: "My",
  users: "Users",
  access: "Access",
  auth: "Auth",
  requests: "Requests",
  login: "Sign in",
};

// fill replaces the ":name" parameters of a path pattern.
function fill(pattern: string, params: Record<string, string> = {}): string {
  return pattern.replace(/:([a-zA-Z]+)/g, (_, key: string) => {
    const value = params[key] ?? "";
    // a ":id" value here means someone is building a router pattern through pathOf, which
    // would escape the colon and register an unreachable route; patternOf is the way.
    if (value.startsWith(":")) {
      throw new Error(`pathOf got the pattern "${value}" for route "${pattern}"; use patternOf`);
    }
    return encodeURIComponent(value);
  });
}

// The registry is compiled once: matching every prefix of a URL per render would
// otherwise rebuild the patterns on each navigation.
const MATCHERS = (Object.keys(ROUTES) as RouteId[]).map((id) => {
  const { path } = ROUTES[id];
  const source = `^${path.replace(/:[a-zA-Z]+/g, "[^/]+")}$`;
  return { id, path, regex: new RegExp(source) };
});

// matchRoute resolves a concrete path to the route metadata that renders it.
export function matchRoute(pathname: string): RouteMeta | undefined {
  const hit = MATCHERS.find((matcher) => matcher.regex.test(pathname));
  return hit ? ROUTES[hit.id] : undefined;
}

// patternOf returns the router pattern for a route, parameters and all. The router
// registers patterns, so a ":id" segment must never pass through fill().
export function patternOf(id: RouteId): string {
  return ROUTES[id].path;
}

// pathOf builds a link from a route id, filling any parameters it declares.
export function pathOf(id: RouteId, params?: Record<string, string>): string {
  return fill(ROUTES[id].path, params);
}

// segmentLabel words one URL segment: a page's label when a route names it, the
// section word when it only groups pages, otherwise the segment itself.
export function segmentLabel(pathname: string): string {
  const meta = matchRoute(pathname);
  if (meta) return meta.label;
  const segment = pathname.slice(pathname.lastIndexOf("/") + 1);
  return (
    PREFIX_LABELS[segment] ?? (/^[0-9a-f-]{8,}$/i.test(segment) ? segment.slice(0, 8) : segment)
  );
}

// isPage tells whether a path renders a page, so a crumb never links to a prefix that only redirects.
export function isPage(pathname: string): boolean {
  return MATCHERS.some((matcher) => matcher.regex.test(pathname));
}

const INSTANCE_TABS = {
  detail: "console.instances.detail",
  tables: "console.instances.tables",
  erd: "console.instances.erd",
  documents: "console.instances.documents",
  keys: "console.instances.keys",
  access: "console.instances.access",
} as const satisfies Record<string, RouteId>;

// PAGE_ROUTES is the ergonomic view of the registry: the same paths, named by UI area.
export const PAGE_ROUTES = {
  CONSOLE,
  INSTANCES: pathOf("console.instances"),
  INSTANCE_DETAIL: (id: string) => pathOf("console.instances.detail", { id }),
  INSTANCE_TAB: (id: string, tab: keyof typeof INSTANCE_TABS) =>
    pathOf(INSTANCE_TABS[tab] ?? "console.instances.detail", { id }),
  MY: "/my",
  MY_PROFILE: pathOf("my.profile"),
  MY_SETTINGS: pathOf("my.settings"),
  MY_ACCESS: pathOf("my.access"),
  ACCESS: pathOf("admin.access"),
  USERS: pathOf("admin.users"),
  USER_DETAIL: (subject: string) => pathOf("admin.users.detail", { subject }),
  LOGIN: pathOf("auth.login"),
  AUTH_CALLBACK: pathOf("auth.callback"),
  SQL: {
    QUERY: pathOf("console.sql.query"),
    ERD: pathOf("console.sql.erd"),
    DML: pathOf("console.sql.dml"),
    DML_DETAIL: (id: string) => pathOf("console.sql.dml.detail", { id }),
    DDL: pathOf("console.sql.ddl"),
    DDL_DETAIL: (id: string) => pathOf("console.sql.ddl.detail", { id }),
  },
  MEMCACHE: {
    KEYS: pathOf("console.memcache"),
  },
  MONGODB: {
    EXPLORE: pathOf("console.mongodb.explore"),
    REQUESTS: pathOf("console.mongodb.requests"),
    REQUEST_NEW: pathOf("console.mongodb.requests.new"),
    REQUEST_DETAIL: (id: string) => pathOf("console.mongodb.requests.detail", { id }),
  },
} as const;

// routePath turns a link path into a router child path (no leading slash).
export const routePath = (path: string) => path.replace(/^\//, "");
