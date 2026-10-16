import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";

// useAction wraps an async action with loading state and error toasts, so
// components stay declarative and no call site repeats try/catch.
export function useAction<TArgs extends unknown[], T>(fn: (...args: TArgs) => Promise<T>) {
  const [loading, setLoading] = useState(false);
  const fnRef = useRef(fn);
  fnRef.current = fn;

  const run = useCallback(async (...args: TArgs): Promise<T | undefined> => {
    setLoading(true);
    try {
      return await fnRef.current(...args);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
      return undefined;
    } finally {
      setLoading(false);
    }
  }, []);

  return { run, loading };
}
