// PAGE_ROUTES centralizes paths shared by the router, navigation, and page links.

export const PAGE_ROUTES = {
  DASHBOARD: "/",
  MY: "/my",
  MY_SETTINGS: "/my/settings",
  MY_ACCESS: "/my/access",
  INSTANCE_DETAIL: (id: string) => `/instances/${id}`,
  INSTANCE_TAB: (id: string, tab: string) => `/instances/${id}/${tab}`,
  ACCESS: "/access",
  USERS: "/users",
  USER_DETAIL: (subject: string) => `/users/${encodeURIComponent(subject)}`,
  SQL: {
    QUERY: "/sql/query",
    ERD: "/sql/erd",
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
