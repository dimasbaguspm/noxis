// PAGE_ROUTES is the single place console page paths are written. The router,
// the navigation registry and every in-page link read from it, so a path is
// defined once and built once. Family groups mirror the engine families.

export const PAGE_ROUTES = {
  DASHBOARD: "/",
  SETTINGS: "/settings",
  PROFILE: "/profile",
  INSTANCE_DETAIL: (id: string) => `/instances/${id}`,
  INSTANCE_DASHBOARD: (id: string) => `/instances/${id}/dashboard`,
  USERS: "/users",
  USER_DETAIL: (subject: string) => `/users/${encodeURIComponent(subject)}`,
  SQL: {
    QUERY: "/sql/query",
    DML: "/sql/dml",
    DML_DETAIL: (id: string) => `/sql/dml/${id}`,
    DDL: "/sql/ddl",
    DDL_DETAIL: (id: string) => `/sql/ddl/${id}`,
  },
  MEMCACHE: {
    KEYS: "/memcache",
  },
  MONGODB: {
    EXPLORE: "/mongodb/explore",
    REQUESTS: "/mongodb/requests",
    REQUEST_NEW: "/mongodb/requests/new",
    REQUEST_DETAIL: (id: string) => `/mongodb/requests/${id}`,
  },
} as const;

// routePath turns a link path into a router child path (no leading slash).
export const routePath = (path: string) => path.replace(/^\//, "");
