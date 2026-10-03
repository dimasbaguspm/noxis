import { useLocation, useParams } from "react-router-dom";

export interface Crumb {
  label: string;
  to?: string;
}

// useBreadcrumbs derives the trail from the route: the shell renders it for
// every page, so no page has to remember to.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { id } = useParams();
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return [{ label: "Dashboard" }];

  const [first, second] = segments;
  switch (first) {
    case "analytics":
      return [{ label: "Dashboard", to: "/" }, { label: "Analytics" }];
    case "query":
      return [{ label: "Dashboard", to: "/" }, { label: "SQL" }, { label: "Query" }];
    case "dml":
    case "ddl": {
      const trail: Crumb[] = [
        { label: "Dashboard", to: "/" },
        { label: "SQL" },
        { label: second ?? first.toUpperCase(), to: `/${first}` },
      ];
      if (id) trail.push({ label: id.slice(0, 8) });
      return trail;
    }
    case "mongo":
      return [{ label: "Dashboard", to: "/" }, { label: "MongoDB" }];
    default:
      return [{ label: "Dashboard", to: "/" }, { label: first }];
  }
}
