import { useMemo, useState } from "react";
import { Network } from "lucide-react";
import { ErdCanvas } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableDetails } from "@/components/sql/table-details";
import { Card, CardContent } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Sheet, SheetBody, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { useErdQuery } from "@/hooks/use-sql";

// InstanceErd is the same graph as /sql/erd without the instance picker: the tab already knows which connection it is.
export function InstanceErd({ connection }: { connection: string }) {
  const erd = useErdQuery(connection);
  const tables = useMemo(() => toErdTables(erd.data), [erd.data]);
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
      <Card className="min-h-0 flex-1 overflow-hidden">
        <CardContent className="h-full min-h-0 p-0">
          {tables.length === 0 ? (
            <Empty icon={Network} title="No tables" description="This instance has no tables to draw." />
          ) : (
            <ErdCanvas key={connection} tables={tables} selectedId={selectedId} query="" onSelect={setSelectedId} />
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
