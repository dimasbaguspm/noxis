import { useMemo, useState } from "react";
import { debounce, group } from "radash";
import { Search, Table2 } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { Table } from "@/lib/api";

// TableBrowser lists the active connection's relations, grouped by schema,
// with a debounced filter. Picking one hands it to the caller.
export function TableBrowser({ tables, onPick }: { tables: Table[]; onPick?: (table: Table) => void }) {
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");

  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (value: string) => setQuery(value)), []);

  const entries = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const matching = needle
      ? tables.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle))
      : tables;
    const groups = group(matching, (t) => t.schema || "public");
    return Object.entries(groups) as [string, Table[]][];
  }, [tables, query]);

  return (
    <div className="flex h-full flex-col gap-2">
      <div className="relative">
        <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-7"
          placeholder="filter tables"
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            void setFilterDebounced(e.target.value);
          }}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        {entries.length === 0 && <p className="px-1 py-2 text-xs text-muted-foreground">no tables</p>}
        {entries.map(([schema, items]) => (
          <div key={schema} className="mb-2">
            <p className="px-1 py-1 text-xs font-medium text-muted-foreground">{schema}</p>
            {items.map((t) => (
              <button
                key={`${t.schema}.${t.name}`}
                onClick={() => onPick?.(t)}
                className={cn(
                  "flex w-full items-center gap-2 rounded px-1.5 py-1 text-left text-sm",
                  "hover:bg-accent hover:text-accent-foreground",
                )}
              >
                <Table2 className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{t.name}</span>
                <span className="ml-auto text-xs text-muted-foreground">{t.columns.length}</span>
              </button>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
