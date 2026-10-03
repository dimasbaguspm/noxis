import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Check, ChevronDown, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { NAV, findNav } from "@/lib/nav";
import { cn } from "@/lib/utils";

// NavMenu is the header's primary navigator: a searchable dropdown that
// replaces the row of nav buttons. It shows the current page, filters the
// registry as you type, and navigates on select.
export function NavMenu() {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const wrap = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);

  const current = findNav(pathname);

  useEffect(() => {
    if (!open) return;
    input.current?.focus();
    function onPointerDown(event: MouseEvent) {
      if (!wrap.current?.contains(event.target as Node)) setOpen(false);
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const groups = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    return NAV.map((group) => ({
      label: group.label,
      items: group.items.filter((item) =>
        needle ? `${group.label ?? ""} ${item.label}`.toLowerCase().includes(needle) : true,
      ),
    })).filter((group) => group.items.length > 0);
  }, [filter]);

  return (
    <div className="relative" ref={wrap}>
      <Button variant="outline" aria-expanded={open} onClick={() => setOpen((v) => !v)} className="gap-2">
        {current.item?.icon ? <current.item.icon className="size-4" /> : <Search className="size-4" />}
        <span>{current.item?.label ?? "Navigate"}</span>
        <ChevronDown className="size-3.5 opacity-60" />
      </Button>

      {open && (
        <div className="absolute left-1/2 z-50 mt-2 w-80 -translate-x-1/2 rounded-lg border border-border bg-popover p-2 text-popover-foreground shadow-lg">
          <div className="relative mb-2">
            <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={input}
              className="pl-7"
              placeholder="search pages (query, dml, redis…)"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>

          <div className="max-h-80 overflow-auto">
            {groups.length === 0 && <p className="px-2 py-3 text-xs text-muted-foreground">no match</p>}
            {groups.map((group, i) => (
              <div key={group.label ?? `group-${i}`} className="mb-1">
                {group.label && (
                  <p className="px-2 py-1 text-[11px] font-medium tracking-wider text-muted-foreground uppercase">
                    {group.label}
                  </p>
                )}
                {group.items.map((item) => {
                  const active = item.to === pathname;
                  return (
                    <button
                      key={item.to}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent hover:text-accent-foreground",
                        active && "bg-accent/60",
                      )}
                      onClick={() => {
                        navigate(item.to);
                        setOpen(false);
                        setFilter("");
                      }}
                    >
                      <item.icon className="size-4 text-muted-foreground" />
                      {item.label}
                      {item.soon && (
                        <Badge variant="outline" className="ml-auto text-[10px]">
                          soon
                        </Badge>
                      )}
                      {active && !item.soon && <Check className="ml-auto size-3.5" />}
                    </button>
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
