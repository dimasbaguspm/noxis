import { LayoutDashboard } from "lucide-react";
import { FAMILIES, type NavItem } from "./families";
import { PAGE_ROUTES } from "./routes";

export type { NavItem };

export interface NavGroup {
  label?: string;
  items: NavItem[];
}

// NAV is the shared source for sidebar, page search, and breadcrumbs.
export const NAV: NavGroup[] = [
  {
    items: [{ to: PAGE_ROUTES.INSTANCES, label: "Instances", icon: LayoutDashboard, end: true }],
  },
  ...FAMILIES.map((family) => ({ label: family.label, items: family.nav })),
];

// findNav resolves the registry entries for a pathname, for the page search palette.
export function findNav(pathname: string): { group?: string; item?: NavItem } {
  for (const group of NAV) {
    for (const item of group.items) {
      if (item.to === pathname) return { group: group.label, item };
    }
  }
  return {};
}
