import type { ComponentType } from "react";
import { Activity, Boxes, Database, Leaf, Network, PenLine } from "lucide-react";
import { DRIVER_PAGES, PAGE_ROUTES, ROUTES, type DriverPage } from "@/constants/routes";
import type { DriverInfo } from "@/lib/api";

// A capability is what a driver's module can actually do. The server declares it per driver, so the console derives its
// pages from the capability and a driver it has never seen still gets the pages its module serves.
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
}

// PAGES is the console's page catalog: one entry per page a capability earns, named by the route that renders it.
const PAGES: Record<Capability, { page: DriverPage; icon: NavItem["icon"] }[]> = {
  [CAPABILITY.SQL]: [
    { page: "query", icon: Database },
    { page: "erd", icon: Network },
    { page: "dml", icon: PenLine },
    { page: "ddl", icon: Activity },
  ],
  [CAPABILITY.KEYS]: [{ page: "keys", icon: Boxes }],
  [CAPABILITY.DOCUMENTS]: [
    { page: "documents", icon: Leaf },
    { page: "requests", icon: PenLine },
  ],
};

const SECTION_META: Record<Capability, { label: string; icon: NavItem["icon"] }> = {
  [CAPABILITY.SQL]: { label: "SQL", icon: Database },
  [CAPABILITY.KEYS]: { label: "Keys", icon: Boxes },
  [CAPABILITY.DOCUMENTS]: { label: "Documents", icon: Leaf },
};

// navItems links each page of a capability under the driver in the URL, so no entry hardcodes a path.
function navItems(capability: Capability, driver: string): NavItem[] {
  return PAGES[capability].map(({ page, icon }) => ({
    to: PAGE_ROUTES.DRIVER(driver, page),
    label: ROUTES[DRIVER_PAGES[page]].label,
    icon,
  }));
}

// capabilitiesOf is the capabilities a driver declares, kept to the ones the console has a page catalog for.
export function capabilitiesOf(catalog: DriverInfo[] | undefined, driver?: string): Capability[] {
  const declared = catalog?.find((entry) => entry.driver === driver)?.capabilities ?? [];
  return declared.filter((capability): capability is Capability =>
    Object.hasOwn(PAGES, capability),
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
  return (first && SECTION_META[first].icon) || Boxes;
}

// pagesOf is the flat navigation of one driver: every page its capabilities earn, in catalog order.
export function pagesOf(catalog: DriverInfo[] | undefined, driver: string): NavItem[] {
  return capabilitiesOf(catalog, driver).flatMap((capability) => navItems(capability, driver));
}

// firstPageOf is the page a driver picker opens: the first page of the first capability the driver declares.
export function firstPageOf(
  catalog: DriverInfo[] | undefined,
  driver: string,
): DriverPage | undefined {
  const capability = capabilitiesOf(catalog, driver)[0];
  return capability ? PAGES[capability][0]?.page : undefined;
}
