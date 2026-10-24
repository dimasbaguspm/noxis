import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Database, Network, Search } from "lucide-react";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ErdCanvas, type ErdTable } from "@/components/sql/erd-canvas";
import { ResultTable } from "@/components/sql/result-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Sheet, SheetBody, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/app/auth-provider";
import { useErdQuery, useTableRowsQuery } from "@/hooks/use-sql";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";
import { compact } from "@/lib/format";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// SqlErdPage is the full-screen schema graph: pick an instance and a table on the left, read and zoom the graph on the right.
export default function SqlErdPage() {
  const { isSuper } = useAuth();
  const { connections, active, busy, setActive } = useConnections(FAMILY.SQL);
  const [params, setParams] = useSearchParams();
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string>();

  const connection = params.get("connection") ?? active ?? connections[0]?.name;
  const erd = useErdQuery(connection);

  const tables = useMemo<ErdTable[]>(
    () =>
      (erd.data?.tables ?? []).map((table) => ({
        id: table.id ?? `${table.schema ?? ""}.${table.name ?? ""}`,
        schema: table.schema ?? "",
        name: table.name ?? "",
        columns: (table.columns ?? []).map((col) => ({
          name: col.name ?? "",
          type: col.type ?? "",
          nullable: col.nullable ?? false,
          primaryKey: col.primaryKey,
          unique: col.unique,
          references: col.references,
        })),
        indexes: (table.indexes ?? []).map((idx) => ({
          name: idx.name ?? "",
          columns: idx.columns ?? [],
          unique: idx.unique,
          primary: idx.primary,
        })),
        estimatedRows: table.estimatedRows,
      })),
    [erd.data],
  );

  const needle = query.trim().toLowerCase();
  const visible = useMemo(
    () =>
      needle
        ? tables.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle))
        : tables,
    [tables, needle],
  );
  const selected = tables.find((t) => t.id === selectedId);

  useEffect(() => {
    setSelectedId(undefined);
  }, [connection]);

  useEffect(() => {
    if (!connection || params.get("connection") === connection) return;
    const next = new URLSearchParams(params);
    next.set("connection", connection);
    setParams(next, { replace: true });
  }, [connection, params, setParams]);

  function pick(name: string) {
    setActive(name);
    const next = new URLSearchParams(params);
    next.set("connection", name);
    setParams(next, { replace: true });
  }

  if (!busy && connections.length === 0) {
    return (
      <Card className="mx-auto mt-8 max-w-lg">
        <CardHeader>
          <CardTitle>No SQL connection found</CardTitle>
          <CardDescription>Register a SQL database to draw its entity relationships.</CardDescription>
        </CardHeader>
        {isSuper && (
          <CardContent>
            <ConnectionDialog trigger="New connection" wizard />
          </CardContent>
        )}
      </Card>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 min-w-0 flex-1">
        <ResizablePanel defaultSize={26} minSize={18}>
          <Card className="mr-2 flex h-full min-h-0 flex-col">
            <CardContent className="flex min-h-0 flex-1 flex-col gap-3">
              <div className="grid gap-1.5">
                <Label>Instance</Label>
                <ConnectionSelect value={connection} onChange={pick} className="w-full" />
              </div>
              <div className="relative">
                <Search className="pointer-events-none absolute top-2 left-2 size-3.5 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Filter tables"
                  className="h-8 pl-7"
                  aria-label="Filter tables"
                />
              </div>
              <div className="min-h-0 flex-1 overflow-auto">
                <div className="grid gap-0.5">
                  {erd.isPending && <Skeleton className="h-24 w-full" />}
                  {!erd.isPending && visible.length === 0 && (
                    <p className="px-2 py-3 text-xs text-muted-foreground">
                      {tables.length === 0 ? "This connection has no tables." : "No table matches this filter."}
                    </p>
                  )}
                  {visible.map((table) => (
                    <button
                      key={table.id}
                      type="button"
                      onClick={() => setSelectedId(table.id)}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent/60",
                        table.id === selectedId && "bg-accent text-accent-foreground",
                      )}
                    >
                      <Network className="size-3.5 shrink-0 text-muted-foreground" />
                      <span className="truncate">{table.name}</span>
                      <span className="ml-auto shrink-0 text-[10px] text-muted-foreground">{table.schema}</span>
                    </button>
                  ))}
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                {tables.length} table(s) · {connections.length} instance(s)
              </p>
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={74} minSize={45} className="min-w-0">
          <Card className="ml-2 flex h-full min-h-0 flex-col overflow-hidden">
            <CardHeader className="shrink-0 border-b">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <CardTitle className="text-base">Entity relationships</CardTitle>
                  <CardDescription>
                    Declared keys and foreign keys of {connection ?? "this instance"}. Nothing is inferred from column names.
                  </CardDescription>
                </div>
                {tables.length > 0 && <Badge variant="secondary">{tables.length} tables</Badge>}
              </div>
            </CardHeader>
            <CardContent className="relative min-h-0 flex-1 p-0">
              {erd.isPending ? (
                <Skeleton className="m-4 h-[60%] w-[calc(100%-2rem)]" />
              ) : erd.error ? (
                <Empty
                  icon={Network}
                  title="The diagram could not load"
                  description={erd.error instanceof Error ? erd.error.message : "the schema request failed"}
                />
              ) : tables.length === 0 ? (
                <Empty
                  icon={Network}
                  title="No tables"
                  description="This instance has no tables to draw."
                />
              ) : (
                <ErdCanvas
                  key={connection}
                  tables={tables}
                  selectedId={selectedId}
                  query={query}
                  onSelect={setSelectedId}
                />
              )}
            </CardContent>
          </Card>
        </ResizablePanel>
      </ResizablePanelGroup>

      <Sheet open={!!selected} onOpenChange={(open) => !open && setSelectedId(undefined)}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{selected?.name}</SheetTitle>
            <SheetDescription>
              {selected?.schema} · {selected?.columns.length ?? 0} column(s)
            </SheetDescription>
          </SheetHeader>
          <SheetBody>
            {selected && <TableDetails table={selected} tables={tables} connection={connection} />}
          </SheetBody>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function TableDetails({ table, tables, connection }: { table: ErdTable; tables: ErdTable[]; connection?: string }) {
  const rows = useTableRowsQuery(connection, table.schema, table.name);
  const incoming = tables.flatMap((t) =>
    t.columns
      .filter((col) => col.references?.table === table.name)
      .map((col) => ({ from: t.name, column: col.name, to: col.references?.column ?? "" })),
  );

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap gap-1.5">
        <Badge variant="secondary">{table.columns.length} columns</Badge>
        {table.estimatedRows !== undefined && <Badge variant="secondary">~{compact(table.estimatedRows)} rows</Badge>}
        <Badge variant="secondary">{table.indexes?.length ?? 0} indexes</Badge>
      </div>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Columns</h3>
        <div className="grid gap-1">
          {table.columns.map((col) => (
            <div key={col.name} className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs">
              <span className="w-5 shrink-0 font-mono text-[10px] text-muted-foreground">
                {col.primaryKey ? "PK" : col.references ? "FK" : col.unique ? "UK" : ""}
              </span>
              <span className="truncate font-mono">{col.name}</span>
              <span className="ml-auto truncate text-muted-foreground">{col.type}</span>
              {col.nullable && <Badge variant="outline">NULL</Badge>}
            </div>
          ))}
        </div>
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Indexes</h3>
        {table.indexes && table.indexes.length > 0 ? (
          <div className="grid gap-1">
            {table.indexes.map((idx) => (
              <div key={idx.name} className="rounded-md border px-2 py-1.5 text-xs">
                <div className="flex items-center gap-1.5">
                  <span className="truncate font-mono">{idx.name}</span>
                  {idx.primary && <Badge variant="outline">primary</Badge>}
                  {idx.unique && !idx.primary && <Badge variant="outline">unique</Badge>}
                </div>
                <div className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">
                  {idx.columns.length > 0 ? idx.columns.join(", ") : "expression"}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No indexes reported.</p>
        )}
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">References</h3>
        {table.columns.some((col) => col.references) ? (
          <div className="grid gap-1">
            {table.columns
              .filter((col) => col.references)
              .map((col) => (
                <div key={col.name} className="rounded-md border px-2 py-1.5 font-mono text-xs">
                  {col.name} → {col.references?.schema}.{col.references?.table}.{col.references?.column}
                </div>
              ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No declared foreign keys.</p>
        )}
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Incoming references</h3>
        {incoming.length > 0 ? (
          <div className="grid gap-1">
            {incoming.map((ref) => (
              <div key={`${ref.from}.${ref.column}`} className="rounded-md border px-2 py-1.5 font-mono text-xs">
                {ref.from}.{ref.column} → {table.name}.{ref.to}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No table references this one.</p>
        )}
      </section>

      <section>
        <h3 className="mb-2 flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase">
          <Database className="size-3.5" /> Preview (100 rows)
        </h3>
        {rows.isPending ? (
          <p className="text-xs text-muted-foreground">Loading rows…</p>
        ) : rows.error ? (
          <p className="text-xs text-destructive">
            {rows.error instanceof Error ? rows.error.message : "the rows could not load"}
          </p>
        ) : (
          <div className="h-64">
            <ResultTable result={rows.data ?? null} title={`${table.schema}.${table.name}`} />
          </div>
        )}
      </section>

      <Button variant="outline" size="sm" render={<Link to={`${PAGE_ROUTES.SQL.QUERY}?connection=${encodeURIComponent(connection ?? "")}`} />}>
        <Network /> Open SQL query
      </Button>
    </div>
  );
}
