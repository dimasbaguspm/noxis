import type { ComponentType } from "react";
import { Activity, Boxes, Database, Leaf, Network, PenLine } from "lucide-react";
import { ROUTES, pathOf, type RouteId } from "@/constants/routes";
import type { DriverInfo } from "@/lib/api";

// A capability is what a driver's module can actually do. The server declares it per driver, so the console groups
// its navigation by capability and a driver it has never seen still appears under the group it serves.
export type Capability = "sql" | "keys" | "documents";

export const CAPABILITY = {
  SQL: "sql",
  KEYS: "keys",
  DOCUMENTS: "documents",
} as const;

export interface NavItem {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  end?: boolean;
  soon?: boolean;
}

export interface Section {
  key: Capability;
  label: string;
  icon: ComponentType<{ className?: string }>;
  // page is the section's landing route.
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

const SECTIONS: Section[] = [
  {
    key: CAPABILITY.SQL,
    label: "SQL",
    icon: Database,
    page: pathOf("console.query"),
    openLabel: "Open in Query",
    nav: navItems([
      { route: "console.query", icon: Database },
      { route: "console.erd", icon: Network },
      { route: "console.changes.dml", icon: PenLine },
      { route: "console.changes.ddl", icon: Activity },
    ]),
  },
  {
    key: CAPABILITY.KEYS,
    label: "Keys",
    icon: Boxes,
    page: pathOf("console.keys"),
    openLabel: "Browse keys",
    nav: navItems([{ route: "console.keys", icon: Boxes, end: true }]),
  },
  {
    key: CAPABILITY.DOCUMENTS,
    label: "Documents",
    icon: Leaf,
    page: pathOf("console.explore"),
    openLabel: "Explore",
    nav: navItems([
      { route: "console.explore", icon: Leaf, end: true },
      { route: "console.requests", icon: PenLine },
    ]),
  },
];

// capabilitiesOf is the capabilities a driver declares, kept to the ones the console has a section for.
export function capabilitiesOf(catalog: DriverInfo[] | undefined, driver?: string): Capability[] {
  const declared = catalog?.find((entry) => entry.driver === driver)?.capabilities ?? [];
  return declared.filter((capability): capability is Capability =>
    SECTIONS.some((section) => section.key === capability),
  );
}

// driverLabel is the word a picker or a sentence uses for a driver: the catalog's label, or the driver string itself
// when the catalog does not know it.
export function driverLabel(catalog: DriverInfo[] | undefined, driver?: string): string {
  return catalog?.find((entry) => entry.driver === driver)?.label ?? driver ?? "";
}

// capabilityIcon is the glyph of a driver's first capability, so a list shows what it serves without naming a
// component per driver; a driver with no known capability falls back to the generic box.
export function capabilityIcon(capabilities: Capability[]): ComponentType<{ className?: string }> {
  const first = capabilities[0];
  return SECTIONS.find((section) => section.key === first)?.icon ?? Boxes;
}

// sections is the navigation the catalog actually supports: a section appears only when some driver declares it.
export function sections(catalog: DriverInfo[] | undefined): Section[] {
  return SECTIONS.filter((section) =>
    (catalog ?? []).some((entry) => entry.capabilities.includes(section.key)),
  );
}
