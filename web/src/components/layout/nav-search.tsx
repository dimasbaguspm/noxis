import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Check, CornerDownLeft, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { NAV, findNav, type NavItem } from "@/lib/nav";
import { cn } from "@/lib/utils";

interface Entry {
  group?: string;
  item: NavItem;
}

const entries: Entry[] = NAV.flatMap((group) => group.items.map((item) => ({ group: group.label, item })));

// NavSearch is the header's navigator: a field that behaves like a button and
// opens a modal palette (⌘K / Ctrl+K). Type to filter, ↑↓ to move, ↵ to go.
export function NavSearch() {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const [cursor, setCursor] = useState(0);
  const list = useRef<HTMLDivElement>(null);

  const current = findNav(pathname);

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
  }, [filter]);

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

  return (
    <Dialog open={open} onOpenChange={change}>
      <DialogTrigger
        render={
          <Button variant="outline" className="w-72 justify-start gap-2 text-muted-foreground font-normal" />
        }
      >
        <Search className="size-4" />
        <span>{current.item?.label ?? "Search pages…"}</span>
        <kbd className="ml-auto rounded border border-border px-1.5 text-[10px] tracking-tight">⌘K</kbd>
      </DialogTrigger>

      <DialogContent showCloseButton={false} className="gap-0 overflow-hidden p-0 sm:max-w-lg">
        <DialogTitle className="sr-only">Navigate</DialogTitle>

        <div className="flex items-center gap-2 border-b border-border px-3 py-2">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <Input
            autoFocus
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              setCursor(0);
            }}
            onKeyDown={onInputKey}
            placeholder="search pages — query, dml, redis…"
            className="h-7 border-0 bg-transparent p-0 shadow-none outline-none placeholder:text-muted-foreground"
          />
          <kbd className="flex items-center gap-1 rounded border border-border px-1.5 text-[10px] text-muted-foreground">
            esc
          </kbd>
        </div>

        <div ref={list} className="max-h-80 overflow-auto p-2">
          {matches.length === 0 && (
            <p className="px-2 py-3 text-sm text-muted-foreground">no page matches “{filter}”</p>
          )}
          {matches.map((entry, i) => {
            const active = entry.item.to === current.item?.to;
            const selected = i === cursor;
            return (
              <Button
                key={entry.item.to}
                variant="ghost"
                size="sm"
                onMouseEnter={() => setCursor(i)}
                onClick={() => go(entry)}
                className={cn(
                  "flex h-auto w-full items-center justify-start gap-2 px-2 py-2 text-left text-sm",
                  selected && "bg-accent text-accent-foreground hover:bg-accent",
                )}
              >
                <entry.item.icon className="size-4 shrink-0 text-muted-foreground" />
                <span>{entry.item.label}</span>
                {entry.group && <span className="text-xs text-muted-foreground">· {entry.group}</span>}
                {entry.item.soon && (
                  <Badge variant="outline" className="text-[10px]">
                    soon
                  </Badge>
                )}
                <span className="ml-auto flex items-center gap-1.5 text-muted-foreground">
                  {active && <Check className="size-3.5" />}
                  {selected && <CornerDownLeft className="size-3.5" />}
                </span>
              </Button>
            );
          })}
        </div>

        <div className="flex items-center gap-3 border-t border-border px-3 py-2 text-[11px] text-muted-foreground">
          <span>↑↓ navigate</span>
          <span>↵ open</span>
          <span>esc close</span>
          <span className="ml-auto">{matches.length} page(s)</span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
