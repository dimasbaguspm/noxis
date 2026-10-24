import { useMemo, useState } from "react";
import { Network } from "lucide-react";
import { ErdCanvas } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableBrowser } from "@/components/sql/table-browser";
import { TableDetails } from "@/components/sql/table-details";
import { Empty } from "@/components/ui/empty";
import { Sheet, SheetBody, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Workspace } from "@/components/layout/workspace";
import { useErdQuery } from "@/hooks/use-sql";
import type { Table } from "@/lib/api";

// InstanceErd is the schema graph with the same table browser as Query on the left and the canvas on the right.
export function InstanceErd({ connection, instanceId }: { connection: string; instanceId?: string }) {
  const erd = useErdQuery(connection);
  const tables = useMemo(() => toErdTables(erd.data), [erd.data]);
  const browserTables = useMemo<Table[]>(
    () =>
      tables.map((table) => ({
        id: table.id,
        schema: table.schema,
        name: table.name,
        columns: table.columns.map((column) => ({
          name: column.name,
          type: column.type,
          nullable: column.nullable,
        })),
      })),
    [tables],
  );
  const [selectedId, setSelectedId] = useState<string>();
  const selected = tables.find((table) => table.id === selectedId);

  return (
    <>
      <div className="flex h-full min-h-0">
        <Workspace
          leftLabel="ERD tables"
          rightLabel="Entity relationship diagram"
          left={
            erd.isPending ? (
              <Skeleton className="h-48 w-full" />
            ) : (
              <TableBrowser tables={browserTables} onSelect={(table) => setSelectedId(table.id)} />
            )
          }
          right={
            erd.isPending ? (
              <Skeleton className="h-64 w-full" />
            ) : erd.error ? (
              <Empty
                icon={Network}
                title="The diagram could not load"
                description={erd.error instanceof Error ? erd.error.message : "the schema request failed"}
              />
            ) : tables.length === 0 ? (
              <Empty icon={Network} title="No tables" description="This instance has no tables to draw." />
            ) : (
              <ErdCanvas key={connection} tables={tables} selectedId={selectedId} query="" onSelect={setSelectedId} />
            )
          }
        />
      </div>

      <Sheet open={!!selected} onOpenChange={(open) => !open && setSelectedId(undefined)}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{selected?.name}</SheetTitle>
            <SheetDescription>
              {selected?.schema} · {selected?.columns.length ?? 0} column(s)
            </SheetDescription>
          </SheetHeader>
          <SheetBody>{selected && <TableDetails table={selected} tables={tables} connection={connection} instanceId={instanceId} />}</SheetBody>
        </SheetContent>
      </Sheet>
    </>
  );
}
