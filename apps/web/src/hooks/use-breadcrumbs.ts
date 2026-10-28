import { useLocation } from "react-router-dom";
import { useCrumbLabels } from "@/app/crumb-provider";
import { isPage, segmentLabel } from "@/constants/routes";

export interface Crumb {
  label: string;
  to?: string;
}

// useBreadcrumbs walks the URL and names every prefix from the route registry, so the trail always matches the page it sits on.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { labels } = useCrumbLabels();
  const segments = pathname.split("/").filter(Boolean);

  const crumbs = segments.map((_, i) => {
    const to = `/${segments.slice(0, i + 1).join("/")}`;
    const leaf = i === segments.length - 1;
    return {
      label: labels[to] ?? segmentLabel(to),
      to: leaf || !isPage(to) ? undefined : to,
    };
  });

  return crumbs.length > 0 ? crumbs : [{ label: "Console" }];
}
