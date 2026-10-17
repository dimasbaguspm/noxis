import { useEffect } from "react";
import { useLocation, useMatches } from "react-router-dom";
import { setPage, startSpan } from "@/lib/tracing";

// routeTemplate turns a concrete path into its route pattern, so every instance
// shares one span name and the collector's cardinality stays low.
function routeTemplate(pathname: string, params: Record<string, string | undefined>): string {
  let out = pathname;
  for (const [key, value] of Object.entries(params)) {
    if (value) out = out.replace(`/${encodeURIComponent(value)}`, `/:${key}`);
  }
  return out;
}

// usePageSpan opens one span per navigation and makes it the parent of the
// actions and API calls the page makes.
export function usePageSpan(): void {
  const { pathname } = useLocation();
  const matches = useMatches();
  const last = matches[matches.length - 1];
  const template = routeTemplate(pathname, last?.params ?? {});

  useEffect(() => {
    const span = startSpan(`page ${template}`);
    setPage(span);
    return () => {
      setPage(undefined);
      span.end();
    };
  }, [template]);
}
