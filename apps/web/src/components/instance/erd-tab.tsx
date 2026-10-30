import { useMemo, useState } from "react";
import { Network } from "lucide-react";
import { ErdCanvas } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableBrowser } from "@/components/sql/table-browser";
import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Workspace } from "@/components/layout/workspace";
import { useErdQuery } from "@/hooks/use-sql";
import type { Table } from "@/lib/api";

// InstanceErd is the schema graph with the same table browser as Query on the left and the canvas on the right.
export function InstanceErd({
  driver,
  connection,
  instanceId,
}: {
  driver: string;
  connection: string;
  instanceId?: string;
}) {
  const erd = useErdQuery(driver, connection);
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

  return (
    <>
      <div className="flex h-full min-h-0">
        <Workspace>
          <Workspace.Pane defaultSize={17} minSize={14}>
            {erd.isPending ? (
              <Skeleton className="h-48 w-full" />
            ) : (
              <TableBrowser tables={browserTables} onSelect={(table) => setSelectedId(table.id)} />
            )}
          </Workspace.Pane>
          <Workspace.Pane defaultSize={83} minSize={40} padded={false}>
            {erd.isPending ? (
              <Skeleton className="m-4 h-64 w-[calc(100%-2rem)]" />
            ) : erd.error ? (
              <Empty
                icon={Network}
                title="The diagram could not load"
                description={
                  erd.error instanceof Error ? erd.error.message : "The schema request failed"
                }
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
                query=""
                layoutKey={instanceId ?? connection}
                onSelect={setSelectedId}
              />
            )}
          </Workspace.Pane>
        </Workspace>
      </div>
    </>
  );
}
