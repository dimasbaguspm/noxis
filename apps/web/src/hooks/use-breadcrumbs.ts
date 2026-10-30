import { useLocation } from "react-router-dom";
import { useCrumbLabels } from "@/app/crumb-provider";
import { useDriversQuery } from "@/hooks/use-config";
import { driverLabel } from "@/lib/console-nav";
import { isPage, segmentLabel } from "@/constants/routes";

export interface Crumb {
  label: string;
  to?: string;
}

// useBreadcrumbs walks the URL and names every prefix from the route registry, so the trail always matches the page it
// sits on; a driver segment is named by the catalog, so the trail reads PostgreSQL rather than postgres.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { labels } = useCrumbLabels();
  const { data: drivers } = useDriversQuery();
  const segments = pathname.split("/").filter(Boolean);

  const crumbs = segments.map((segment, i) => {
    const to = `/${segments.slice(0, i + 1).join("/")}`;
    const leaf = i === segments.length - 1;
    const driver = (drivers ?? []).some((entry) => entry.driver === segment)
      ? driverLabel(drivers, segment)
      : undefined;
    return {
      label: labels[to] ?? driver ?? segmentLabel(to),
      to: leaf || !isPage(to) ? undefined : to,
    };
  });

  return crumbs.length > 0 ? crumbs : [{ label: "Console" }];
}
