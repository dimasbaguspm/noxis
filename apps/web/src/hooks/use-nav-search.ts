import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useDriver } from "@/hooks/use-driver";
import { navGroups, type NavItem } from "@/lib/nav";

interface Entry {
  group?: string;
  item: NavItem;
}

// useNavSearch is the palette's state: open, the filter, the highlighted row, the ⌘K listener and the pick that navigates.
// The entries come from the driver catalog, so the palette lists exactly the sections the sidebar shows.
export function useNavSearch() {
  const navigate = useNavigate();
  const { driver, drivers } = useDriver();
  const groups = useMemo(() => navGroups(drivers, driver), [drivers, driver]);
  const entries = useMemo<Entry[]>(
    () => groups.flatMap((group) => group.items.map((item) => ({ group: group.label, item }))),
    [groups],
  );
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const [cursor, setCursor] = useState(0);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen(true);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // opening starts from an empty filter and the first row, so the palette never opens on a stale match.
  function change(next: boolean) {
    setOpen(next);
    if (next) {
      setFilter("");
      setCursor(0);
    }
  }

  const matches = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    if (!needle) return entries;
    return entries.filter(({ group, item }) =>
      `${group ?? ""} ${item.label} ${item.to}`.toLowerCase().includes(needle),
    );
  }, [entries, filter]);

  function go(entry: Entry | undefined) {
    if (!entry) return;
    navigate(entry.item.to);
    change(false);
  }

  function onInputKey(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setCursor((c) => Math.min(c + 1, matches.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setCursor((c) => Math.max(c - 1, 0));
    } else if (event.key === "Enter") {
      event.preventDefault();
      go(matches[cursor]);
    }
  }

  // typing restarts the highlight so Enter always opens the top match.
  function search(value: string) {
    setFilter(value);
    setCursor(0);
  }

  return { open, change, filter, search, cursor, setCursor, matches, groups, go, onInputKey };
}
