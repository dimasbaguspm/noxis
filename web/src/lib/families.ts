import type { ComponentType } from "react";
import { Activity, BarChart3, Boxes, Database, Leaf, PenLine } from "lucide-react";

// The engine-family registry: one place that names every family, its landing
// page, the API namespace it is served under and its navigation entries. The
// sidebar, the palette, the breadcrumbs and the connection state all read from
// here, so adding a family means adding one entry, not editing four lists.

export type FamilyKey = "sql" | "memcache" | "mongodb";

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
    key: "sql",
    label: "SQL",
    icon: Database,
    page: "/sql/query",
    openLabel: "Open in Query",
    nav: [
      { to: "/sql/query", label: "Query", icon: Database },
      { to: "/sql/dml", label: "DML", icon: PenLine },
      { to: "/sql/ddl", label: "DDL", icon: Activity },
      { to: "/sql/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    key: "memcache",
    label: "Memory Cache",
    icon: Boxes,
    page: "/memcache",
    openLabel: "Browse keys",
    nav: [
      { to: "/memcache", label: "Keys", icon: Boxes, end: true },
      { to: "/memcache/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    key: "mongodb",
    label: "MongoDB",
    icon: Leaf,
    page: "/mongodb",
    openLabel: "Explore",
    nav: [
      { to: "/mongodb", label: "Explore", icon: Leaf, end: true },
      { to: "/mongodb/requests", label: "Requests", icon: PenLine },
      { to: "/mongodb/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
];

export const FAMILY = Object.fromEntries(FAMILIES.map((f) => [f.key, f])) as Record<FamilyKey, Family>;
