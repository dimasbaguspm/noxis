import type { ComponentType } from "react";
import { Activity, Boxes, Database, Leaf, Network, PenLine } from "lucide-react";
import { ROUTES, pathOf, type RouteId } from "@/constants/routes";

// FAMILY is the console's own grouping of the engines it serves. It is a UI vocabulary only: the API
// addresses an engine by driver, and DRIVER_FAMILY maps each driver onto the group it is shown under.
export const FAMILY = {
  SQL: "sql",
  MEMCACHE: "memcache",
  MONGODB: "mongodb",
} as const;

export type FamilyKey = (typeof FAMILY)[keyof typeof FAMILY];

// DRIVER is the wire vocabulary the console needs when no connection is in hand. Every other call takes
// the driver from the connection it acts on (a postgres instance is "postgres", not "sql").
export const DRIVER = {
  MONGO: "mongo",
  // SQL_QUEUE addresses the SQL change queue when no single connection is in scope: that route only picks
  // the SQL engine, and every SQL driver serves the same queue.
  SQL_QUEUE: "postgres",
} as const;

// DRIVER_FAMILY maps the server's drivers onto the console's groups, so the sidebar keeps speaking families
// while every request speaks drivers.
const DRIVER_FAMILY: Record<string, FamilyKey> = {
  postgres: FAMILY.SQL,
  mysql: FAMILY.SQL,
  mariadb: FAMILY.SQL,
  sqlite: FAMILY.SQL,
  cockroach: FAMILY.SQL,
  redis: FAMILY.MEMCACHE,
  valkey: FAMILY.MEMCACHE,
  mongo: FAMILY.MONGODB,
};

// familyOfDriver resolves the group a driver is shown under; an unknown driver has no group rather than a wrong one.
export function familyOfDriver(driver?: string): FamilyKey | undefined {
  return driver ? DRIVER_FAMILY[driver] : undefined;
}

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
