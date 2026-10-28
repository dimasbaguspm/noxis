import { useState } from "react";

export type SortKey = "name" | "type" | "ttl" | "size";

interface Selection {
  scope: string;
  keys: string[];
}

const NO_SELECTION: Selection = { scope: "", keys: [] };

// useMemcacheBrowser keeps the key table's selection, its sort column and the empty-database dialog; a selection only counts while its scope matches.
export function useMemcacheBrowser(scope: string) {
  const [selection, setSelection] = useState<Selection>(NO_SELECTION);
  const selected = selection.scope === scope ? selection.keys : [];
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "name", desc: false });
  const [flushOpen, setFlushOpen] = useState(false);

  // dropping the selection records the current scope, so a later scope change also starts empty.
  function clearSelection() {
    setSelection({ scope, keys: [] });
  }

  function toggle(key: string) {
    const keys = selected.includes(key)
      ? selected.filter((item) => item !== key)
      : [...selected, key];
    setSelection({ scope, keys });
  }

  // sorting on the active column flips the direction; a different column starts ascending.
  function sortBy(key: SortKey) {
    setSort((prev) => ({ key, desc: prev.key === key ? !prev.desc : false }));
  }

  return { selected, sort, sortBy, clearSelection, toggle, flushOpen, setFlushOpen };
}
