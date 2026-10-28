import { useLocation, useParams } from "react-router-dom";
import { findNav } from "@/lib/nav";
import { PAGE_ROUTES } from "@/lib/routes";

export interface Crumb {
  label: string;
  to?: string;
}

// useBreadcrumbs derives group/page labels from NAV and falls back to the parent for nested routes.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { id } = useParams();

  if (pathname === PAGE_ROUTES.INSTANCES) return [{ label: "Instances" }];

  // the instance tabs live under the console's instance list
  if (pathname.startsWith(`${PAGE_ROUTES.INSTANCES}/`)) {
    return [{ label: "Instances", to: PAGE_ROUTES.INSTANCES }, { label: "Instance detail" }];
  }

  const segments = pathname.split("/").filter(Boolean);
  const direct = findNav(pathname);
  const target = direct.item ? pathname : `/${segments.slice(0, 2).join("/")}`;
  const { group, item } = direct.item ? direct : findNav(target);

  const crumbs: Crumb[] = [{ label: "Instances", to: PAGE_ROUTES.INSTANCES }];
  if (group) crumbs.push({ label: group });
  if (item) crumbs.push({ label: item.label, to: direct.item ? undefined : target });
  else crumbs.push({ label: segments.at(-1) ?? "" });
  if (!direct.item && id) {
    // a detail page gets its section name instead of the raw id hash
    const leaf =
      segments[1] === "sql" && segments[2] === "dml" ? "DML Detail" :
      segments[1] === "sql" && segments[2] === "ddl" ? "DDL Detail" :
      segments[1] === "mongodb" && segments[2] === "requests" ? "Request Detail" :
      id.slice(0, 8);
    crumbs.push({ label: leaf });
  }
  return crumbs;
}
