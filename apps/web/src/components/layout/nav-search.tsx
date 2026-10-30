import { useRef } from "react";
import { useLocation } from "react-router-dom";
import { Check, CornerDownLeft, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { findNav } from "@/lib/nav";
import { useNavSearch } from "@/hooks/use-nav-search";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

// NavSearch is the header's navigator: a field that behaves like a button and opens a modal palette (⌘K / Ctrl+K).
export function NavSearch() {
  const { pathname } = useLocation();
  const { open, change, filter, search, cursor, setCursor, matches, groups, go, onInputKey } =
    useNavSearch();
  const list = useRef<HTMLDivElement>(null);

  const current = findNav(groups, pathname);

  return (
    <Dialog open={open} onOpenChange={change}>
      <DialogTrigger
        render={
          <Button
            variant="outline"
            className="w-auto justify-start gap-2 font-normal text-muted-foreground sm:w-72"
          >
            <Search className="size-4" />
            <span className="hidden sm:inline">{current.item?.label ?? "Search pages…"}</span>
            <kbd className="ml-auto hidden rounded border border-border px-1.5 text-[10px] tracking-tight sm:inline">
              ⌘K
            </kbd>
          </Button>
        }
      />

      <DialogContent showCloseButton={false} className="gap-0 overflow-hidden p-0 sm:max-w-lg">
        <DialogTitle className="sr-only">Navigate</DialogTitle>

        <div className="flex items-center gap-2 border-b border-border px-3 py-2">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <Input
            autoFocus
            value={filter}
            onChange={(e) => search(e.target.value)}
            onKeyDown={onInputKey}
            placeholder="Search pages — Query, DML, Redis…"
            className="h-7 border-0 bg-transparent p-0 shadow-none outline-none placeholder:text-muted-foreground"
          />
          <kbd className="flex items-center gap-1 rounded border border-border px-1.5 text-[10px] text-muted-foreground">
            Esc
          </kbd>
        </div>

        <div ref={list} className="max-h-80 overflow-auto p-2">
          {matches.length === 0 && (
            <P size="sm" className="px-2 py-3">
              no page matches “{filter}”
            </P>
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
                {entry.group && (
                  <span className="text-xs text-muted-foreground">· {entry.group}</span>
                )}
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
          <span>Esc close</span>
          <span className="ml-auto">{matches.length} page(s)</span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
