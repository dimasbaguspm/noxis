import type { ComponentType } from "react";
import {
  Activity,
  BarChart3,
  Boxes,
  Database,
  LayoutDashboard,
  Leaf,
  PenLine,
  TerminalSquare,
} from "lucide-react";

export interface NavItem {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  end?: boolean;
  soon?: boolean;
}

export interface NavGroup {
  label?: string;
  items: NavItem[];
}

// One registry for the whole navigation: the sidebar, the header menu and the
// breadcrumbs all read it, so a page only has to be added here once.
export const NAV: NavGroup[] = [
  {
    items: [{ to: "/", label: "Dashboard", icon: LayoutDashboard, end: true }],
  },
  {
    label: "SQL",
    items: [
      { to: "/query", label: "Query", icon: Database },
      { to: "/dml", label: "DML", icon: PenLine },
      { to: "/ddl", label: "DDL", icon: Activity },
      { to: "/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    label: "MongoDB",
    items: [
      { to: "/mongo/explore", label: "Explore", icon: Leaf, soon: true },
      { to: "/mongo/crud", label: "CRUD Request", icon: PenLine, soon: true },
      { to: "/mongo/analytics", label: "Analytics", icon: BarChart3, soon: true },
    ],
  },
  {
    label: "Redis",
    items: [
      { to: "/redis/explore", label: "Explore", icon: Boxes, soon: true },
      { to: "/redis/console", label: "Console", icon: TerminalSquare, soon: true },
      { to: "/redis/analytics", label: "Analytics", icon: BarChart3, soon: true },
    ],
  },
];

// Placeholder copy for the engines that are scoped but not built yet: the
// pages exist so the shape of the product is visible and honest.
export interface PlaceholderSpec {
  title: string;
  summary: string;
  planned: string[];
  icon: ComponentType<{ className?: string }>;
}

export const PLACEHOLDERS: Record<string, PlaceholderSpec> = {
  "/mongo/explore": {
    title: "MongoDB · Explore",
    summary: "Browse databases and collections the way the SQL sidebar browses schemas and tables.",
    planned: [
      "connection registry reuse (host, credentials, replica set)",
      "collection list with document counts and indexes",
      "document viewer with a query filter bar (find/aggregate)",
      "export the current result as JSON",
    ],
    icon: Leaf,
  },
  "/mongo/crud": {
    title: "MongoDB · CRUD Request",
    summary: "Document changes go through the same review flow as SQL DML.",
    planned: [
      "request shape: instance, notes, operation, filter/document, rollback",
      "validation dry-run against a temporary collection copy",
      "approve → apply with the affected-document estimate recorded",
      "artifact trail: nothing is ever deleted",
    ],
    icon: PenLine,
  },
  "/mongo/analytics": {
    title: "MongoDB · Analytics",
    summary: "Live instance metrics for a replica set, streamed like the SQL analytics page.",
    planned: [
      "connections, op counters (query/insert/update/delete) and their rates",
      "memory: resident, virtual, mapped and cache usage",
      "replication lag and oplog window",
      "slow-query sampling (profiler) with a per-collection breakdown",
    ],
    icon: BarChart3,
  },
  "/redis/explore": {
    title: "Redis · Explore",
    summary: "A key browser for Redis: scan by pattern, inspect values, TTLs and encodings.",
    planned: [
      "SCAN-based browsing (never KEYS) with pattern filter and cursors",
      "value inspector per type: string, hash, list, set, zset, stream",
      "TTL editing and safe deletion with confirmation",
      "keyspace summary per database index",
    ],
    icon: Boxes,
  },
  "/redis/console": {
    title: "Redis · Console",
    summary: "Run Redis commands with the same guard rails the SQL console uses.",
    planned: [
      "read-only commands (GET, HGETALL, SCAN, INFO, TTL…) run directly",
      "write commands (SET/DEL/EXPIRE/HSET…) go through the review flow",
      "command hints with arity and complexity",
      "result rendered per type instead of a raw reply",
    ],
    icon: TerminalSquare,
  },
  "/redis/analytics": {
    title: "Redis · Analytics",
    summary: "INFO-driven metrics, streamed over SSE like the SQL analytics page.",
    planned: [
      "memory: used, peak, fragmentation ratio, evicted keys",
      "throughput: ops/sec, hit ratio, expired vs evicted",
      "connections: clients, blocked, rejected",
      "persistence: RDB/AOF status, last save, replication offsets",
    ],
    icon: BarChart3,
  },
};

// findNav resolves the registry entries for a pathname, for breadcrumbs.
export function findNav(pathname: string): { group?: string; item?: NavItem } {
  for (const group of NAV) {
    for (const item of group.items) {
      if (item.to === pathname) return { group: group.label, item };
    }
  }
  return {};
}
