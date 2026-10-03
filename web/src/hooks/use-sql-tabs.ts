import { useCallback, useEffect, useMemo, useState } from "react";

export interface SqlTab {
  id: string;
  title: string;
  sql: string;
}

const STORAGE_KEY = "noXis.sql.tabs";
const DEFAULT_SQL = "SELECT 1";

// useSqlTabs keeps the SQL editor's tabs: at least one always exists, there is
// no upper bound, and the whole strip (tabs + active id) is persisted so a
// reload lands where you left off.
export function useSqlTabs() {
  const initial = useMemo(load, []);
  const [tabs, setTabs] = useState<SqlTab[]>(initial.tabs);
  const [activeId, setActiveId] = useState<string>(initial.activeId);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ tabs, activeId }));
    } catch {
      /* private mode: tabs simply are not remembered */
    }
  }, [tabs, activeId]);

  const active = tabs.find((tab) => tab.id === activeId) ?? tabs[0];

  const add = useCallback(
    (sql = "") => {
      const id = crypto.randomUUID();
      setTabs((prev) => [...prev, { id, title: nextTitle(prev), sql: sql || DEFAULT_SQL }]);
      setActiveId(id);
    },
    [],
  );

  const update = useCallback(
    (sql: string) => {
      setTabs((prev) => prev.map((tab) => (tab.id === (activeId ?? tab.id) ? { ...tab, sql } : tab)));
    },
    [activeId],
  );

  const close = useCallback(
    (id: string) => {
      setTabs((prev) => {
        if (prev.length === 1) return prev; // one tab is the floor
        const next = prev.filter((tab) => tab.id !== id);
        if (id === activeId) setActiveId(next[next.length - 1].id);
        return next;
      });
    },
    [activeId],
  );

  return { tabs, active, activeId: active?.id ?? "", add, update, close, select: setActiveId };
}

// nextTitle numbers tabs continuously ("Query 3") without re-using numbers.
function nextTitle(tabs: SqlTab[]): string {
  const highest = tabs.reduce((max, tab) => {
    const n = Number(tab.title.replace(/\D/g, ""));
    return Number.isFinite(n) && n > max ? n : max;
  }, 0);
  return `Query ${highest + 1}`;
}

function load(): { tabs: SqlTab[]; activeId: string } {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as { tabs?: SqlTab[]; activeId?: string };
      if (parsed.tabs?.length) {
        const activeId = parsed.activeId && parsed.tabs.some((t) => t.id === parsed.activeId)
          ? parsed.activeId
          : parsed.tabs[0].id;
        return { tabs: parsed.tabs, activeId };
      }
    }
  } catch {
    /* fall through to a fresh tab */
  }
  const tab: SqlTab = { id: crypto.randomUUID(), title: "Query 1", sql: DEFAULT_SQL };
  return { tabs: [tab], activeId: tab.id };
}
