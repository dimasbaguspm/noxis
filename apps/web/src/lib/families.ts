import type { ComponentType } from "react";
import { Activity, Boxes, Database, Leaf, Network, PenLine } from "lucide-react";
import { ROUTES, pathOf, type RouteId } from "@/constants/routes";

// FAMILY is the family key vocabulary. The strings are the API Namespace segment too, so they are written once here.
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

// navItems names each entry once: the path and the label both come from the route registry.
function navItems(entries: { route: RouteId; icon: NavItem["icon"]; end?: boolean }[]): NavItem[] {
  return entries.map((entry) => ({
    to: pathOf(entry.route),
    label: ROUTES[entry.route].label,
    icon: entry.icon,
    end: entry.end,
  }));
}

export const FAMILIES: Family[] = [
  {
    key: FAMILY.SQL,
    label: "SQL",
    icon: Database,
    page: pathOf("console.sql.query"),
    openLabel: "Open in Query",
    nav: navItems([
      { route: "console.sql.query", icon: Database },
      { route: "console.sql.erd", icon: Network },
      { route: "console.sql.dml", icon: PenLine },
      { route: "console.sql.ddl", icon: Activity },
    ]),
  },
  {
    key: FAMILY.MEMCACHE,
    label: "Memory Cache",
    icon: Boxes,
    page: pathOf("console.memcache"),
    openLabel: "Browse keys",
    nav: navItems([{ route: "console.memcache", icon: Boxes, end: true }]),
  },
  {
    key: FAMILY.MONGODB,
    label: "MongoDB",
    icon: Leaf,
    page: pathOf("console.mongodb.explore"),
    openLabel: "Explore",
    nav: navItems([
      { route: "console.mongodb.explore", icon: Leaf, end: true },
      { route: "console.mongodb.requests", icon: PenLine },
    ]),
  },
];

// familyByKey resolves a family descriptor, for the cross-family surfaces (dashboard, new-connection wizard).
export const familyByKey = FAMILIES.reduce(
  (acc, family) => ({ ...acc, [family.key]: family }),
  {} as Record<FamilyKey, Family>,
);
