import { useMemo, useState } from "react";
import { debounce, group } from "radash";
import { ChevronDown, ChevronRight, EllipsisVertical, ListTree, PlayCircle, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import type { Table } from "@/lib/api";

interface Props {
  tables: Table[];
  onPick?: (table: Table) => void;
  onRecords?: (table: Table) => void;
  onAnalyze?: (table: Table) => void;
}

// TableBrowser: schemas first, then their tables. Rows expand to columns and
// carry per-table actions.
export function TableBrowser({ tables, onPick, onRecords, onAnalyze }: Props) {
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [schemaFilter, setSchemaFilter] = useState<string | null>(null);

  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const schemas = useMemo(() => Object.keys(group(tables, (t) => t.schema || "public")).sort(), [tables]);

  const entries = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const scope = schemaFilter ? tables.filter((t) => (t.schema || "public") === schemaFilter) : tables;
    const matching = needle ? scope.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle)) : scope;
    const groups = group(matching, (t) => t.schema || "public");
    return Object.entries(groups) as [string, Table[]][];
  }, [tables, query, schemaFilter]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex flex-wrap gap-1">
        <Button size="xs" variant={schemaFilter === null ? "secondary" : "ghost"} onClick={() => setSchemaFilter(null)}>
          all schemas
        </Button>
        {schemas.map((s) => (
          <Button
            key={s}
            size="xs"
            variant={schemaFilter === s ? "secondary" : "ghost"}
            onClick={() => setSchemaFilter(s)}
          >
            {s}
          </Button>
        ))}
      </div>

      <div className="relative">
        <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-7"
          placeholder="search tables"
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            void setFilterDebounced(e.target.value);
          }}
        />
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {entries.length === 0 && <p className="px-1 py-2 text-xs text-muted-foreground">no tables</p>}
        {entries.map(([schema, items]) => (
          <div key={schema} className="mb-2">
            <p className="flex items-center gap-1 px-1 py-1 text-xs font-medium text-muted-foreground">
              <ListTree className="size-3" />
              {schema}
            </p>
            {items.map((t) => {
              const key = `${t.schema}.${t.name}`;
              const open = expanded === key;
              return (
                <div key={key}>
                  <div
                    className={cn(
                      "group flex items-center gap-1 rounded px-1.5 py-1 text-sm",
                      "hover:bg-accent hover:text-accent-foreground",
                    )}
                  >
                    <button
                      className="flex min-w-0 flex-1 items-center gap-1 text-left"
                      onClick={() => setExpanded(open ? null : key)}
                    >
                      {open ? (
                        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />
                      )}
                      <span className="truncate">{t.name}</span>
                      <span className="ml-1 text-xs text-muted-foreground">{t.columns.length}</span>
                    </button>
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={<Button size="icon" variant="ghost" aria-label={`actions for ${t.name}`} />}
                      >
                        <EllipsisVertical />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => onPick?.(t)}>
                          <PlayCircle /> Use in query
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onRecords?.(t)}>
                          <ListTree /> Show records (limit 100)
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onAnalyze?.(t)}>Analyze</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  {open && (
                    <div className="mb-1 ml-5 border-l border-border pl-2">
                      {t.columns.map((col) => (
                        <p key={col.name} className="flex items-center gap-2 py-0.5 text-xs">
                          <span className="font-medium">{col.name}</span>
                          <span className="text-muted-foreground">{col.type}</span>
                          {col.nullable && <span className="text-muted-foreground/70">null</span>}
                        </p>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ))}
      </ScrollArea>
    </div>
  );
}
