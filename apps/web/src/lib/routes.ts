// CONSOLE is the base path of every database engine page, so the URL says which product section the user is in.
// Each future section (object storage, containers, config) gets a base of its own next to it.
export const CONSOLE = "/console";

// PAGE_ROUTES centralizes paths shared by the router, navigation, and page links.
export const PAGE_ROUTES = {
  CONSOLE,
  INSTANCES: `${CONSOLE}/instances`,
  INSTANCE_DETAIL: (id: string) => `${CONSOLE}/instances/${id}`,
  INSTANCE_TAB: (id: string, tab: string) => `${CONSOLE}/instances/${id}/${tab}`,
  MY: "/my",
  MY_PROFILE: "/my/profile",
  MY_SETTINGS: "/my/settings",
  MY_ACCESS: "/my/access",
  ACCESS: "/access",
  USERS: "/users",
  USER_DETAIL: (subject: string) => `/users/${encodeURIComponent(subject)}`,
  LOGIN: "/login",
  AUTH_CALLBACK: "/auth/callback",
  SQL: {
    QUERY: `${CONSOLE}/sql/query`,
    ERD: `${CONSOLE}/sql/erd`,
    DML: `${CONSOLE}/sql/dml`,
    DML_DETAIL: (id: string) => `${CONSOLE}/sql/dml/${id}`,
    DDL: `${CONSOLE}/sql/ddl`,
    DDL_DETAIL: (id: string) => `${CONSOLE}/sql/ddl/${id}`,
  },
  MEMCACHE: {
    KEYS: `${CONSOLE}/memcache`,
  },
  MONGODB: {
    EXPLORE: `${CONSOLE}/mongodb/explore`,
    REQUESTS: `${CONSOLE}/mongodb/requests`,
    REQUEST_NEW: `${CONSOLE}/mongodb/requests/new`,
    REQUEST_DETAIL: (id: string) => `${CONSOLE}/mongodb/requests/${id}`,
  },
} as const;

// routePath turns a link path into a router child path (no leading slash).
export const routePath = (path: string) => path.replace(/^\//, "");
