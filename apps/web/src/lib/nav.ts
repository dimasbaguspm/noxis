import { LayoutDashboard } from "lucide-react";
import { ROUTES, pathOf } from "@/constants/routes";
import { driverLabel, pagesOf, type NavItem } from "@/lib/console-nav";
import type { DriverInfo } from "@/lib/api";

export type { NavItem };

export interface NavGroup {
  label?: string;
  items: NavItem[];
}

// navGroups is the shared source for the sidebar and the page search palette: the instances link plus one group per
// driver, so the palette reaches every engine rather than only the one in the URL.
export function navGroups(catalog: DriverInfo[] | undefined): NavGroup[] {
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
    ...(catalog ?? []).map((entry) => ({
      label: driverLabel(catalog, entry.driver),
      items: pagesOf(catalog, entry.driver),
    })),
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
