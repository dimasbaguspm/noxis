import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Network, Search } from "lucide-react";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ErdCanvas, type ErdTable } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableDetails } from "@/components/sql/table-details";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Sheet, SheetBody, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/app/auth-provider";
import { useErdQuery } from "@/hooks/use-sql";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";
import { cn } from "@/lib/utils";

// SqlErdPage is the full-screen schema graph: pick an instance and a table on the left, read and zoom the graph on the right.
export default function SqlErdPage() {
  const { isSuper } = useAuth();
  const { connections, active, busy, setActive } = useConnections(FAMILY.SQL);
  const [params, setParams] = useSearchParams();
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string>();

  const wanted = params.get("instance") ?? params.get("connection");
  const picked = connections.find((c) => c.id === wanted || c.name === wanted);
  const connection = picked?.name ?? active ?? connections[0]?.name;
  const erd = useErdQuery(connection);

  const tables = useMemo<ErdTable[]>(() => toErdTables(erd.data), [erd.data]);

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
    const id = connections.find((c) => c.name === connection)?.id;
    if (!id || params.get("instance") === id) return;
    const next = new URLSearchParams(params);
    next.delete("connection");
    next.set("instance", id);
    setParams(next, { replace: true });
  }, [connection, connections, params, setParams]);

  function pick(name: string) {
    setActive(name);
    const next = new URLSearchParams(params);
    next.delete("connection");
    const id = connections.find((c) => c.name === name)?.id;
    if (id) next.set("instance", id);
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
        <ResizablePanel defaultSize={20} minSize={16}>
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
