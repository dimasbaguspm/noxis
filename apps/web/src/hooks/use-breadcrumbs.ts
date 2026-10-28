import { useLocation } from "react-router-dom";
import { title } from "radash";
import { useCrumbLabels } from "@/app/crumb-provider";
import { PAGE_ROUTES } from "@/lib/routes";

export interface Crumb {
  label: string;
  to?: string;
}

// SEGMENT_LABELS words the path segments a breadcrumb can show; an unknown segment falls back to its own short form.
const SEGMENT_LABELS: Record<string, string> = {
  console: "Console",
  instances: "Instances",
  sql: "SQL",
  query: "Query",
  erd: "ERD",
  dml: "DML",
  ddl: "DDL",
  memcache: "Memory Cache",
  mongodb: "MongoDB",
  explore: "Explore",
  requests: "Requests",
  new: "New request",
  tables: "Tables",
  documents: "Documents",
  keys: "Keys",
  access: "Access",
  my: "My",
  profile: "Profile",
  settings: "Settings",
  users: "Users",
};

// LINKABLE is the set of paths that render a page, so a crumb never points at a prefix that only redirects.
const LINKABLE = new Set<string>([
  PAGE_ROUTES.MY,
  PAGE_ROUTES.MY_PROFILE,
  PAGE_ROUTES.MY_SETTINGS,
  PAGE_ROUTES.MY_ACCESS,
  PAGE_ROUTES.ACCESS,
  PAGE_ROUTES.USERS,
  PAGE_ROUTES.SQL.QUERY,
  PAGE_ROUTES.SQL.ERD,
  PAGE_ROUTES.SQL.DML,
  PAGE_ROUTES.SQL.DDL,
  PAGE_ROUTES.MEMCACHE.KEYS,
  PAGE_ROUTES.MONGODB.EXPLORE,
  PAGE_ROUTES.MONGODB.REQUESTS,
]);

// segmentLabel words one segment, or trims an id hash to something readable.
function segmentLabel(segment: string): string {
  return SEGMENT_LABELS[segment] ?? (/^[0-9a-f-]{8,}$/i.test(segment) ? segment.slice(0, 8) : title(segment));
}

// useBreadcrumbs walks the URL and names every prefix, so the trail always matches the page it sits on.
export function useBreadcrumbs(): Crumb[] {
  const { pathname } = useLocation();
  const { labels } = useCrumbLabels();
  const segments = pathname.split("/").filter(Boolean);

  const crumbs = segments.map((segment, i) => {
    const to = `/${segments.slice(0, i + 1).join("/")}`;
    const leaf = i === segments.length - 1;
    return {
      label: labels[to] ?? segmentLabel(segment),
      to: leaf || !LINKABLE.has(to) ? undefined : to,
    };
  });

  return crumbs.length > 0 ? crumbs : [{ label: "Console" }];
}
