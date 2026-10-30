import { Link, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { PAGE_ROUTES } from "@/constants/routes";
import { useEffect, useMemo, useState } from "react";
import { Network } from "lucide-react";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ErdCanvas, type ErdTable } from "@/components/sql/erd-canvas";
import { toErdTables } from "@/components/sql/erd-data";
import { TableBrowser } from "@/components/sql/table-browser";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { FilterField } from "@/components/filters/filter-field";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/app/auth-provider";
import { useAnalyzeMutation, useErdQuery } from "@/hooks/use-sql";
import type { Table } from "@/lib/api";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { Workspace } from "@/components/layout/workspace";

// SqlErdPage is the full-screen schema graph: pick an instance and a table on the left, read and zoom the graph on the right.
export default function SqlErdPage() {
  const { isSuper, canAccess } = useAuth();
  const navigate = useNavigate();
  // the driver comes from the URL and the instance from ?instance=, so the page cannot reach another driver's schema
  const { connections, connection, driver, busy, pick } = useDriverConnections();
  const [selectedId, setSelectedId] = useState<string>();

  const picked = connections.find((c) => c.name === connection);
  const erd = useErdQuery(driver, connection);
  const analyze = useAnalyzeMutation();

  const tables = useMemo<ErdTable[]>(() => toErdTables(erd.data), [erd.data]);
  const schemaTables = (erd.data?.tables ?? []) as unknown as Table[];
  // the row menu refreshes the planner's statistics; it is a write, so it is only offered with write access
  const onAnalyze = (table: Table) =>
    driver && connection
      ? analyze.mutate({ driver, connection, table: `${table.schema}.${table.name}` })
      : undefined;

  useEffect(() => {
    setSelectedId(undefined);
  }, [connection]);

  if (!busy && connections.length === 0) {
    return (
      <Card className="mx-auto mt-8 max-w-lg">
        <CardHeader>
          <CardTitle>No SQL connection found</CardTitle>
          <CardDescription>
            Register a SQL database to draw its entity relationships.
          </CardDescription>
        </CardHeader>
        {isSuper && (
          <CardContent>
            <Button render={<Link to={`${PAGE_ROUTES.INSTANCE_CREATE}?driver=${driver ?? ""}`} />}>
              New connection
            </Button>
          </CardContent>
        )}
      </Card>
    );
  }

  return (
    <>
      <div className="flex h-full min-h-0">
        <Workspace>
          <Workspace.Pane defaultSize={17} minSize={14}>
            <div className="flex min-h-0 flex-1 flex-col gap-4">
              <FilterField label="Instance">
                <ConnectionSelect
                  connections={connections}
                  value={connection}
                  onChange={pick}
                  className="w-full"
                />
              </FilterField>
              <div className="min-h-0 flex-1">
                <TableBrowser
                  tables={schemaTables}
                  onRecords={(t) =>
                    navigate(
                      `${PAGE_ROUTES.DRIVER(driver, "query")}?sql=${encodeURIComponent(
                        `SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`,
                      )}`,
                    )
                  }
                  onSelect={(t) => setSelectedId(t.id ?? `${t.schema ?? ""}.${t.name ?? ""}`)}
                  onAnalyze={canAccess(connection, "write") ? onAnalyze : undefined}
                />
              </div>
            </div>
          </Workspace.Pane>
          <Workspace.Pane defaultSize={83} minSize={40} padded={false}>
            {erd.isPending ? (
              <Skeleton className="m-4 h-[60%] w-[calc(100%-2rem)]" />
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
                layoutKey={picked?.id ?? connection}
                onSelect={setSelectedId}
              />
            )}
          </Workspace.Pane>
        </Workspace>
      </div>
    </>
  );
}
