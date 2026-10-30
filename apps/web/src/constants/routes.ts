// Route metadata is the single source for the router, the sidebar, the breadcrumb, the document title, and the cache keys.

export type RouteSection = "driver" | "my" | "admin" | "auth";

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

// DRIVERS is the base of every engine page: the driver is a path segment, because the engine a page acts on is the one
// fact a console URL must carry. Instances stay at the root, since one instance is addressed by id, not by driver.
export const DRIVERS = "/drivers";

export const ROUTES = {
  "driver.query": {
    path: `${DRIVERS}/:driver/query`,
    title: "Query",
    label: "Query",
    entity: "query",
    section: "driver",
  },
  "driver.erd": {
    path: `${DRIVERS}/:driver/erd`,
    title: "ERD",
    label: "ERD",
    entity: "erd",
    section: "driver",
  },
  "driver.changes.dml": {
    path: `${DRIVERS}/:driver/changes/dml`,
    title: "DML changes",
    label: "DML",
    entity: "changes",
    section: "driver",
  },
  "driver.changes.dml.detail": {
    path: `${DRIVERS}/:driver/changes/dml/:id`,
    title: "DML change",
    label: "DML change",
    entity: "changes",
    section: "driver",
  },
  "driver.changes.ddl": {
    path: `${DRIVERS}/:driver/changes/ddl`,
    title: "DDL changes",
    label: "DDL",
    entity: "changes",
    section: "driver",
  },
  "driver.changes.ddl.detail": {
    path: `${DRIVERS}/:driver/changes/ddl/:id`,
    title: "DDL change",
    label: "DDL change",
    entity: "changes",
    section: "driver",
  },
  "driver.keys": {
    path: `${DRIVERS}/:driver/keys`,
    title: "Keys",
    label: "Keys",
    entity: "keys",
    section: "driver",
  },
  "driver.documents": {
    path: `${DRIVERS}/:driver/documents`,
    title: "Documents",
    label: "Documents",
    entity: "explore",
    section: "driver",
  },
  "driver.requests": {
    path: `${DRIVERS}/:driver/requests`,
    title: "Requests",
    label: "Requests",
    entity: "requests",
    section: "driver",
  },
  "driver.requests.new": {
    path: `${DRIVERS}/:driver/requests/new`,
    title: "New request",
    label: "New request",
    entity: "requests",
    section: "driver",
  },
  "driver.requests.detail": {
    path: `${DRIVERS}/:driver/requests/:id`,
    title: "Request",
    label: "Request",
    entity: "requests",
    section: "driver",
  },
  "console.instances": {
    path: "/instances",
    title: "Instances",
    label: "Instances",
    entity: "instances",
    section: "driver",
  },
  "console.instances.create": {
    path: "/instances/create",
    title: "New instance",
    label: "New instance",
    entity: "instances",
    section: "driver",
  },
  "console.instances.detail": {
    path: "/instances/:id",
    title: "Instance",
    label: "Instance",
    entity: "instances",
    section: "driver",
  },
  "console.instances.tables": {
    path: "/instances/:id/tables",
    title: "Tables",
    label: "Tables",
    entity: "instances",
    section: "driver",
  },
  "console.instances.erd": {
    path: "/instances/:id/erd",
    title: "ERD",
    label: "ERD",
    entity: "instances",
    section: "driver",
  },
  "console.instances.documents": {
    path: "/instances/:id/documents",
    title: "Documents",
    label: "Documents",
    entity: "instances",
    section: "driver",
  },
  "console.instances.keys": {
    path: "/instances/:id/keys",
    title: "Keys",
    label: "Keys",
    entity: "instances",
    section: "driver",
  },
  "console.instances.access": {
    path: "/instances/:id/access",
    title: "Access",
    label: "Access",
    entity: "access",
    section: "driver",
  },
  "console.instances.dml": {
    path: "/instances/:id/changes/dml",
    title: "DML changes",
    label: "DML",
    entity: "changes",
    section: "driver",
  },
  "console.instances.ddl": {
    path: "/instances/:id/changes/ddl",
    title: "DDL changes",
    label: "DDL",
    entity: "changes",
    section: "driver",
  },
  "console.instances.requests": {
    path: "/instances/:id/requests",
    title: "Requests",
    label: "Requests",
    entity: "requests",
    section: "driver",
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
  drivers: "Drivers",
  instances: "Instances",
  changes: "Changes",
  requests: "Requests",
  my: "My",
  users: "Users",
  access: "Access",
  auth: "Auth",
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
  const names: string[] = [];
  const source = path.replace(/:([a-zA-Z]+)/g, (_, name: string) => {
    names.push(name);
    return "([^/]+)";
  });
  return { id, names, regex: new RegExp(`^${source}$`) };
});

export interface RouteMatch {
  id: RouteId;
  meta: RouteMeta;
  // params are the decoded path parameters, so a page reads its driver and id from the URL alone.
  params: Record<string, string>;
}

// matchPath resolves a concrete path to its route and the parameters the URL filled.
export function matchPath(pathname: string): RouteMatch | undefined {
  for (const matcher of MATCHERS) {
    const hit = matcher.regex.exec(pathname);
    if (!hit) continue;
    const params: Record<string, string> = {};
    matcher.names.forEach((name, index) => {
      params[name] = decodeURIComponent(hit[index + 1] ?? "");
    });
    return { id: matcher.id, meta: ROUTES[matcher.id], params };
  }
  return undefined;
}

// matchRoute resolves a concrete path to the route metadata that renders it.
export function matchRoute(pathname: string): RouteMeta | undefined {
  return matchPath(pathname)?.meta;
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

// DRIVER_PAGES names the page a driver can offer; the capability decides which of them the sidebar shows.
export const DRIVER_PAGES = {
  query: "driver.query",
  erd: "driver.erd",
  dml: "driver.changes.dml",
  ddl: "driver.changes.ddl",
  keys: "driver.keys",
  documents: "driver.documents",
  requests: "driver.requests",
} as const satisfies Record<string, RouteId>;

export type DriverPage = keyof typeof DRIVER_PAGES;

const INSTANCE_TABS = {
  detail: "console.instances.detail",
  tables: "console.instances.tables",
  erd: "console.instances.erd",
  documents: "console.instances.documents",
  keys: "console.instances.keys",
  access: "console.instances.access",
  // the review tabs keep the queue's own leaf so the tab id and the path suffix agree
  "changes/dml": "console.instances.dml",
  "changes/ddl": "console.instances.ddl",
  requests: "console.instances.requests",
} as const satisfies Record<string, RouteId>;

// PAGE_ROUTES is the ergonomic view of the registry: the same paths, named by UI area.
export const PAGE_ROUTES = {
  INSTANCES: pathOf("console.instances"),
  INSTANCE_CREATE: pathOf("console.instances.create"),
  INSTANCE_DETAIL: (id: string) => pathOf("console.instances.detail", { id }),
  INSTANCE_TAB: (id: string, tab: keyof typeof INSTANCE_TABS) =>
    pathOf(INSTANCE_TABS[tab] ?? "console.instances.detail", { id }),
  // DRIVER is the one builder for an engine page, so no call site composes a driver path by hand.
  DRIVER: (driver: string, page: DriverPage) => pathOf(DRIVER_PAGES[page], { driver }),
  CHANGE_DETAIL: (driver: string, kind: "dml" | "ddl", id: string) =>
    pathOf(kind === "dml" ? "driver.changes.dml.detail" : "driver.changes.ddl.detail", {
      driver,
      id,
    }),
  REQUEST_NEW: (driver: string) => pathOf("driver.requests.new", { driver }),
  REQUEST_DETAIL: (driver: string, id: string) => pathOf("driver.requests.detail", { driver, id }),
  MY: "/my",
  MY_PROFILE: pathOf("my.profile"),
  MY_SETTINGS: pathOf("my.settings"),
  MY_ACCESS: pathOf("my.access"),
  ACCESS: pathOf("admin.access"),
  USERS: pathOf("admin.users"),
  USER_DETAIL: (subject: string) => pathOf("admin.users.detail", { subject }),
  LOGIN: pathOf("auth.login"),
  AUTH_CALLBACK: pathOf("auth.callback"),
} as const;

// routePath turns a link path into a router child path (no leading slash).
export const routePath = (path: string) => path.replace(/^\//, "");
