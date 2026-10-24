import { useMemo, useState } from "react";
import { debounce, group, title } from "radash";
import { Activity, ChevronDown, ChevronRight, ListTree, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RowMenu } from "@/components/ui/row-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { Table } from "@/lib/api";
import { P } from "@/components/ui/typography";

interface Props {
  tables: Table[];
  onRecords?: (table: Table) => void;
  onAnalyze?: (table: Table) => void;
  onSelect?: (table: Table) => void;
}

// TableBrowser: schemas first, then their tables.
export function TableBrowser({ tables, onRecords, onAnalyze, onSelect }: Props) {
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [selectedSchema, setSelectedSchema] = useState("");

  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const schemas = useMemo(() => Object.keys(group(tables, (t) => t.schema || "public")).sort(), [tables]);
  const schemaFilter = schemas.includes(selectedSchema) ? selectedSchema : "";

  const entries = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const scope = schemaFilter ? tables.filter((t) => (t.schema || "public") === schemaFilter) : tables;
    const matching = needle ? scope.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle)) : scope;
    const groups = group(matching, (t) => t.schema || "public");
    return Object.entries(groups) as [string, Table[]][];
  }, [tables, query, schemaFilter]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="grid gap-1.5">
        <Label>Schema</Label>
        <Select value={schemaFilter || "all"} onValueChange={(next) => setSelectedSchema(String(next) === "all" ? "" : String(next))}>
        <SelectTrigger className="w-full" aria-label="schema">
          <SelectValue placeholder="All schemas" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All schemas</SelectItem>
          {schemas.map((s) => (
            <SelectItem key={s} value={s}>
              {s}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      </div>

      <div className="grid gap-1.5">
        <Label>Search tables</Label>
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
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {entries.length === 0 && <P className="px-1 py-2">{title("no tables")}</P>}
        {entries.map(([schema, items]) => (
          <div key={schema} className="mb-2">
            <P className="flex items-center gap-1 px-1 py-1 font-medium">
              <ListTree className="size-3" />
              {schema}
            </P>
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
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex min-w-0 flex-1 items-center gap-1 justify-start px-1"
                      onClick={() => {
                        setExpanded(open ? null : key);
                        onSelect?.(t);
                      }}
                    >
                      {open ? (
                        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />
                      )}
                      <span className="truncate">{t.name}</span>
                      <span className="ml-1 text-xs text-muted-foreground">{t.columns.length}</span>
                    </Button>
                    {(onRecords || onAnalyze) && <RowMenu
                      label={`actions for ${t.name}`}
                      items={[
                        {
                          label: "Show Records",
                          icon: <ListTree className="size-3.5" />,
                          onSelect: () => onRecords?.(t),
                        },
                        // ANALYZE is a write: only offered when the caller holds it
                        ...(onAnalyze
                          ? [
                              {
                                label: "Analyze",
                                icon: <Activity className="size-3.5" />,
                                onSelect: () => onAnalyze(t),
                              },
                            ]
                          : []),
                      ]}
                    />}
                  </div>
                  {open && (
                    <ul className="mb-1 ml-5 grid gap-0.5 border-l border-border py-1 pl-2">
                      {t.columns.map((col) => (
                        <li key={col.name} className="flex items-center justify-between gap-2 text-xs">
                          <span className="truncate font-mono">{col.name}</span>
                          <span
                            className={cn(
                              "shrink-0 rounded border px-1 text-[10px]",
                              col.nullable
                                ? "border-border text-muted-foreground"
                                : "border-transparent text-muted-foreground/50",
                            )}
                          >
                            {title(col.nullable ? "nullable" : "required")}
                          </span>
                        </li>
                      ))}
                    </ul>
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
