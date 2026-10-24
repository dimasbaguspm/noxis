import { useCallback, useState } from "react";

export interface SqlTab {
  id: string;
  title: string;
  sql: string;
}

const DEFAULT_SQL = "SELECT 1";

function newTab(title: string, sql = DEFAULT_SQL): SqlTab {
  return { id: crypto.randomUUID(), title, sql };
}

// useSqlTabs keeps the SQL editor's tabs: at least one always exists, there is no upper bound, and every visit starts fresh on a single "Query 1" tab.
export function useSqlTabs() {
  const [state, setState] = useState(() => {
    const first = newTab("Query 1");
    return { tabs: [first], activeId: first.id };
  });

  const active = state.tabs.find((tab) => tab.id === state.activeId) ?? state.tabs[0];

  const add = useCallback((sql = "") => {
    setState((prev) => {
      const tab = newTab(nextTitle(prev.tabs), sql || DEFAULT_SQL);
      return { tabs: [...prev.tabs, tab], activeId: tab.id };
    });
  }, []);

  const update = useCallback((sql: string) => {
    setState((prev) => ({
      ...prev,
      tabs: prev.tabs.map((tab) => (tab.id === prev.activeId ? { ...tab, sql } : tab)),
    }));
  }, []);

  const close = useCallback((id: string) => {
    setState((prev) => {
      if (prev.tabs.length === 1) return prev; // one tab is the floor
      const tabs = prev.tabs.filter((tab) => tab.id !== id);
      const activeId = id === prev.activeId ? tabs[tabs.length - 1].id : prev.activeId;
      return { tabs, activeId };
    });
  }, []);

  const select = useCallback((id: string) => setState((prev) => ({ ...prev, activeId: id })), []);

  return { tabs: state.tabs, active, activeId: active?.id ?? "", add, update, close, select };
}

// nextTitle numbers tabs continuously ("Query 3") without re-using numbers.
function nextTitle(tabs: SqlTab[]): string {
  const highest = tabs.reduce((max, tab) => {
    const n = Number(tab.title.replace(/\D/g, ""));
    return Number.isFinite(n) && n > max ? n : max;
  }, 0);
  return `Query ${highest + 1}`;
}
