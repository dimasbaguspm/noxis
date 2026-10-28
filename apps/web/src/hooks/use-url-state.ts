import { useCallback, useMemo } from "react";
import { useSearchParams } from "react-router-dom";

// useUrlState makes filters shareable and back/forward aware; define defaults at module scope to keep them stable.
export function useUrlState<T extends Record<string, string>>(defaults: T) {
  const [params, setParams] = useSearchParams();

  const state = useMemo(() => {
    const out = { ...defaults };
    for (const key of Object.keys(defaults) as (keyof T)[]) {
      const raw = params.get(key as string);
      if (raw !== null) out[key] = raw as T[keyof T];
    }
    return out;
  }, [params, defaults]);

  const patch = useCallback(
    (updates: Partial<T>) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          for (const [key, value] of Object.entries(updates)) {
            const fallback = (defaults as Record<string, string>)[key];
            if (value !== undefined && value !== fallback) next.set(key, value as string);
            else next.delete(key);
          }
          return next;
        },
        { replace: true },
      );
    },
    [setParams, defaults],
  );

  return [state, patch] as const;
}
