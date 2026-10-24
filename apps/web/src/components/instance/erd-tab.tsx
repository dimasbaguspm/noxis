import { useMemo, useState } from "react";
import { Network, Search } from "lucide-react";
import { ErdCanvas } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableDetails } from "@/components/sql/table-details";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Sheet, SheetBody, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { useErdQuery } from "@/hooks/use-sql";

// InstanceErd is the same graph as /sql/erd without the instance picker: the tab already knows which connection it is.
export function InstanceErd({ connection }: { connection: string }) {
  const erd = useErdQuery(connection);
  const tables = useMemo(() => toErdTables(erd.data), [erd.data]);
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string>();
  const selected = tables.find((t) => t.id === selectedId);

  if (erd.isPending) return <Skeleton className="h-64 w-full" />;
  if (erd.error) {
    return (
      <Empty
        icon={Network}
        title="The diagram could not load"
        description={erd.error instanceof Error ? erd.error.message : "the schema request failed"}
      />
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute top-2 left-2 size-3.5 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter tables"
            className="h-8 pl-7"
            aria-label="Filter tables"
          />
        </div>
        <Badge variant="secondary">{tables.length} tables</Badge>
      </div>

      <Card className="min-h-0 flex-1 overflow-hidden">
        <CardContent className="h-full min-h-0 p-0">
          {tables.length === 0 ? (
            <Empty icon={Network} title="No tables" description="This instance has no tables to draw." />
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

      <Sheet open={!!selected} onOpenChange={(open) => !open && setSelectedId(undefined)}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{selected?.name}</SheetTitle>
            <SheetDescription>
              {selected?.schema} · {selected?.columns.length ?? 0} column(s)
            </SheetDescription>
          </SheetHeader>
          <SheetBody>{selected && <TableDetails table={selected} tables={tables} connection={connection} />}</SheetBody>
        </SheetContent>
      </Sheet>
    </div>
  );
}
