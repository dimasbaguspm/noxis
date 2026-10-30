import { LayoutDashboard } from "lucide-react";
import { ROUTES, pathOf } from "@/constants/routes";
import { sections, type NavItem } from "@/lib/console-nav";
import type { DriverInfo } from "@/lib/api";

export type { NavItem };

export interface NavGroup {
  label?: string;
  items: NavItem[];
}

// navGroups is the shared source for the sidebar and the page search palette: the instances link plus one group per
// capability, linked under the driver in the URL; the paths and labels are the route registry's.
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
    ...sections(catalog, driver).map((section) => ({ label: section.label, items: section.nav })),
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
