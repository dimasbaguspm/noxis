// PAGE_ROUTES is the single place console page paths are written. The router,
// the navigation registry and every in-page link read from it, so a path is
// defined once and built once. Family groups mirror the engine families.

export const PAGE_ROUTES = {
  DASHBOARD: "/",
  SQL: {
    QUERY: "/sql/query",
    DML: "/sql/dml",
    DML_DETAIL: (id: string) => `/sql/dml/${id}`,
    DDL: "/sql/ddl",
    DDL_DETAIL: (id: string) => `/sql/ddl/${id}`,
    ANALYTICS: "/sql/analytics",
  },
  MEMCACHE: {
    KEYS: "/memcache",
    ANALYTICS: "/memcache/analytics",
  },
  MONGODB: {
    EXPLORE: "/mongodb",
    REQUESTS: "/mongodb/requests",
    REQUEST_NEW: "/mongodb/requests/new",
    REQUEST_DETAIL: (id: string) => `/mongodb/requests/${id}`,
    ANALYTICS: "/mongodb/analytics",
  },
} as const;

// LEGACY_SQL holds the pre-family SQL URLs. The router redirects them to
// PAGE_ROUTES.SQL so bookmarks and old links keep working.
export const LEGACY_SQL = {
  QUERY: "query",
  DML: "dml",
  DML_DETAIL: "dml/:id",
  DDL: "ddl",
  DDL_DETAIL: "ddl/:id",
  ANALYTICS: "analytics",
} as const;

// routePath turns a link path into a router child path (no leading slash).
export const routePath = (path: string) => path.replace(/^\//, "");
