import { useCallback } from "react";
import { trackAction } from "@/lib/tracing";

// useTrackEvent wraps a user action in a span, so the API calls it makes join
// the page's trace. Every call is a no-op when browser tracing is off.
export function useTrackEvent() {
  return useCallback(trackAction, []);
}
