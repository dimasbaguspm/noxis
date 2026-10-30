import { LayoutDashboard } from "lucide-react";
import { ROUTES, pathOf } from "@/constants/routes";
import { driverLabel, pagesOf, type NavItem } from "@/lib/console-nav";
import type { DriverInfo } from "@/lib/api";

export type { NavItem };

export interface NavGroup {
  label?: string;
  items: NavItem[];
}

// navGroups is the shared source for the sidebar and the page search palette: the instances link plus the pages the
// driver in the URL supports, named after that driver; the paths and labels are the route registry's.
export function navGroups(catalog: DriverInfo[] | undefined, driver: string): NavGroup[] {
  return [
    {
      items: [
        {
          to: pathOf("console.instances"),
          label: ROUTES["console.instances"].label,
          icon: LayoutDashboard,
          end: true,
        },
      ],
    },
    { label: driverLabel(catalog, driver), items: pagesOf(catalog, driver) },
  ];
}

// findNav resolves the registry entries for a pathname, for the page search palette.
export function findNav(groups: NavGroup[], pathname: string): { group?: string; item?: NavItem } {
  for (const group of groups) {
    for (const item of group.items) {
      if (item.to === pathname) return { group: group.label, item };
    }
  }
  return {};
}
