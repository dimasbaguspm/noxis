import { useEffect, useState } from "react";
import { debounce } from "radash";

// useDebounce returns a value once it has been stable for `delay` ms, so a fast typist triggers one update instead of one per keystroke.
export function useDebounce<T>(value: T, delay = 200): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const update = debounce({ delay }, (v: T) => setDebounced(v));
    update(value);
    return update.cancel;
  }, [value, delay]);
  return debounced;
}
