import type { ComponentType } from "react";
import {
  Activity,
  BarChart3,
  Boxes,
  Database,
  LayoutDashboard,
  Leaf,
  PenLine,
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
      { to: "/mongodb", label: "Explore", icon: Leaf, end: true },
      { to: "/mongodb/requests", label: "Requests", icon: PenLine },
      { to: "/mongodb/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    label: "Memory Cache",
    items: [
      { to: "/memcache", label: "Keys", icon: Boxes, end: true },
      { to: "/memcache/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
];

// findNav resolves the registry entries for a pathname, for breadcrumbs.
export function findNav(pathname: string): { group?: string; item?: NavItem } {
  for (const group of NAV) {
    for (const item of group.items) {
      if (item.to === pathname) return { group: group.label, item };
    }
  }
  return {};
}
