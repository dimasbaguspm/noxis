import { useLocation, useParams } from "react-router-dom";
import { findNav } from "@/lib/nav";

export interface Crumb {
  label: string;
  to?: string;
}

// useBreadcrumbs derives the trail from the navigation registry: an exact
// match gives group → page, and a nested route (a change detail) falls back
// to its parent page plus the short id.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { id } = useParams();

  if (pathname === "/") return [{ label: "Dashboard" }];

  // instances are one stable page under the dashboard's instance list
  if (pathname.startsWith("/instances/")) {
    return [{ label: "Dashboard", to: "/" }, { label: "Instances" }, { label: "Instance detail" }];
  }

  const segments = pathname.split("/").filter(Boolean);
  const direct = findNav(pathname);
  const target = direct.item ? pathname : `/${segments.slice(0, 2).join("/")}`;
  const { group, item } = direct.item ? direct : findNav(target);

  const crumbs: Crumb[] = [{ label: "Dashboard", to: "/" }];
  if (group) crumbs.push({ label: group });
  if (item) crumbs.push({ label: item.label, to: direct.item ? undefined : target });
  else crumbs.push({ label: segments.at(-1) ?? "" });
  if (!direct.item && id) crumbs.push({ label: id.slice(0, 8) });
  return crumbs;
}
