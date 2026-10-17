import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { beginTrace, endSpan, endTrace, startSpan } from "@/lib/tracing";

// useRouteTrace opens one trace per navigation: every API call the screen makes
// joins it, so a single trace covers the page and the backend work it causes.
export function useRouteTrace(): void {
  const { pathname } = useLocation();
  useEffect(() => {
    const traceId = beginTrace();
    const span = startSpan(`navigation ${pathname}`, traceId);
    return () => {
      endSpan(span);
      endTrace(traceId);
    };
  }, [pathname]);
}
