import { useCallback, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { NAV } from "@/lib/nav";

const STORAGE_KEY = "noxis.nav.groups";
const HOME = "_home";

function groupKey(label?: string): string {
  return label ?? HOME;
}

// A group folds away when it has nothing to do yet (every item is "soon").
function defaultOpen(label?: string): boolean {
  const group = NAV.find((g) => g.label === label);
  if (!group) return true;
  return !group.items.every((item) => item.soon);
}

function load(): Record<string, boolean> {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Record<string, boolean>) : {};
  } catch {
    return {};
  }
}

// useNavGroups drives the collapsible sidebar sections: the choice persists,
// the group holding the current page is always unfolded, and the next engine
// families start folded because there is nothing to do in them yet.
export function useNavGroups() {
  const { pathname } = useLocation();
  const [state, setState] = useState<Record<string, boolean>>(load);

  const isOpen = useCallback(
    (label?: string) => state[groupKey(label)] ?? defaultOpen(label),
    [state],
  );

  const toggle = useCallback((label?: string) => {
    const key = groupKey(label);
    setState((prev) => ({ ...prev, [key]: !(prev[key] ?? defaultOpen(label)) }));
  }, []);

  useEffect(() => {
    const active = NAV.find((group) => group.items.some((item) => item.to === pathname));
    if (active && active.label && !(state[groupKey(active.label)] ?? defaultOpen(active.label))) {
      setState((prev) => ({ ...prev, [groupKey(active.label)]: true }));
    }
  }, [pathname, state]);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      /* private mode: folding is simply not remembered */
    }
  }, [state]);

  return { isOpen, toggle };
}
