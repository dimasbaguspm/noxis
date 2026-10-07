import type { ComponentType } from "react";
import { Activity, BarChart3, Boxes, Database, Leaf, PenLine } from "lucide-react";
import { PAGE_ROUTES } from "./routes";

// The engine-family registry: one place that names every family, its landing
// page, the API namespace it is served under and its navigation entries. The
// sidebar, the palette, the breadcrumbs and the connection state all read from
// here, so adding a family means adding one entry, not editing four lists.

// FAMILY is the family key vocabulary. The strings are the API Namespace
// segment too, so they are written once here.
export const FAMILY = {
  SQL: "sql",
  MEMCACHE: "memcache",
  MONGODB: "mongodb",
} as const;

export type FamilyKey = (typeof FAMILY)[keyof typeof FAMILY];

export interface NavItem {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  end?: boolean;
  soon?: boolean;
}

export interface Family {
  key: FamilyKey;
  label: string;
  icon: ComponentType<{ className?: string }>;
  // page is the family's landing route, used by the dashboard's detail action.
  page: string;
  openLabel: string;
  nav: NavItem[];
}

export const FAMILIES: Family[] = [
  {
    key: FAMILY.SQL,
    label: "SQL",
    icon: Database,
    page: PAGE_ROUTES.SQL.QUERY,
    openLabel: "Open in Query",
    nav: [
      { to: PAGE_ROUTES.SQL.QUERY, label: "Query", icon: Database },
      { to: PAGE_ROUTES.SQL.DML, label: "DML", icon: PenLine },
      { to: PAGE_ROUTES.SQL.DDL, label: "DDL", icon: Activity },
      { to: PAGE_ROUTES.SQL.ANALYTICS, label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    key: FAMILY.MEMCACHE,
    label: "Memory Cache",
    icon: Boxes,
    page: PAGE_ROUTES.MEMCACHE.KEYS,
    openLabel: "Browse keys",
    nav: [
      { to: PAGE_ROUTES.MEMCACHE.KEYS, label: "Keys", icon: Boxes, end: true },
      { to: PAGE_ROUTES.MEMCACHE.ANALYTICS, label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    key: FAMILY.MONGODB,
    label: "MongoDB",
    icon: Leaf,
    page: PAGE_ROUTES.MONGODB.EXPLORE,
    openLabel: "Explore",
    nav: [
      { to: PAGE_ROUTES.MONGODB.EXPLORE, label: "Explore", icon: Leaf, end: true },
      { to: PAGE_ROUTES.MONGODB.REQUESTS, label: "Requests", icon: PenLine },
      { to: PAGE_ROUTES.MONGODB.ANALYTICS, label: "Analytics", icon: BarChart3 },
    ],
  },
];

// familyByKey resolves a family descriptor, for the cross-family surfaces
// (dashboard, new-connection wizard).
export const familyByKey = FAMILIES.reduce(
  (acc, family) => ({ ...acc, [family.key]: family }),
  {} as Record<FamilyKey, Family>,
);
