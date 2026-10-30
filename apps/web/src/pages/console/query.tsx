import { ConnectionSelect } from "@/components/connection/connection-select";
import { ResultSummary, ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlTabs } from "@/components/sql/sql-tabs";
import { TableBrowser } from "@/components/sql/table-browser";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { FilterField } from "@/components/filters/filter-field";
import { Workspace } from "@/components/layout/workspace";
import { Link, useSearchParams } from "react-router-dom";
import { PAGE_ROUTES } from "@/constants/routes";
import { useAuth } from "@/app/auth-provider";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useAnalyzeMutation, useConnectionSchemaQuery, useRunQueryMutation } from "@/hooks/use-sql";
import { useSqlTabs } from "@/hooks/use-sql-tabs";
import { classify } from "@/lib/statements";
import { Play, Database } from "lucide-react";

export default function QueryPage() {
  const { canAccess, isSuper } = useAuth();
  const [params] = useSearchParams();
  const { connection, driver, connections, busy, pick } = useDriverConnections();

  const tabs = useSqlTabs(params.get("sql") ?? undefined);
  const tables = useConnectionSchemaQuery(driver, connection).data ?? [];
  const run = useRunQueryMutation();
  const readResult = run.data;
  const loading = run.isPending;
  const execute = (sql: string) => {
    if (!driver || !connection || !sql.trim()) return;
    void run.mutateAsync({ driver, connection, sql });
  };

  const analyze = useAnalyzeMutation();
  const analyzed = analyze.data;
  const onAnalyze = (table: { schema: string; name: string }) =>
    driver && connection
      ? analyze.mutate({ driver, connection, table: `${table.schema}.${table.name}` })
      : undefined;

  const sql = tabs.active?.sql ?? "";
  const canRun = !!driver && !!connection && sql.trim().length > 0;
  const kind = classify(sql);
  const result = analyzed ?? readResult ?? null;

  // No SQL connection to query: keep the workspace and explain it in the pane, instead of collapsing to a card.
  if (connections.length === 0) {
    return (
      <div className="flex h-full min-h-0">
        <Workspace>
          <Workspace.Pane defaultSize={17} minSize={14}>
            <FilterField label="Instance">
              <ConnectionSelect connections={[]} value="" onChange={() => {}} className="w-full" />
            </FilterField>
          </Workspace.Pane>
          <Workspace.Pane defaultSize={83} minSize={40}>
            <Empty
              className="flex-1"
              icon={Database}
              title={busy ? "Loading connections…" : "No connection found"}
              description={
                busy
                  ? "Fetching your registered SQL databases."
                  : "Register a SQL database to start browsing and querying."
              }
              action={
                !busy && isSuper ? (
                  <Button
                    render={<Link to={`${PAGE_ROUTES.INSTANCE_CREATE}?driver=${driver ?? ""}`} />}
                  >
                    New connection
                  </Button>
                ) : undefined
              }
            />
          </Workspace.Pane>
        </Workspace>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0">
      <Workspace footer={result ? <ResultSummary result={result} /> : undefined}>
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
                tables={tables}
                onRecords={(t) => tabs.add(`SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`)}
                onAnalyze={canAccess(connection, "write") ? onAnalyze : undefined}
              />
            </div>
          </div>
        </Workspace.Pane>

        <Workspace.Split defaultSize={83} minSize={40} direction="vertical">
          <Workspace.Pane defaultSize={45} minSize={20}>
            <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">
              <SqlTabs
                tabs={tabs.tabs}
                activeId={tabs.activeId}
                onSelect={tabs.select}
                onClose={tabs.close}
                onAdd={() => tabs.add()}
              />
              <SqlEditor
                value={sql}
                onChange={tabs.update}
                tables={tables}
                height="100%"
                placeholder="SELECT …"
                onRun={() => {
                  if (canRun) void execute(sql);
                }}
              />
              <div className="flex items-center gap-2">
                <Button onClick={() => void execute(sql)} disabled={!canRun} aria-busy={loading}>
                  <Play /> Run
                </Button>
                <Button
                  variant="outline"
                  onClick={() => void execute(`EXPLAIN ${sql}`)}
                  disabled={!canRun || kind !== "read"}
                >
                  Explain
                </Button>
                <span className="text-xs text-muted-foreground">⌘/Ctrl + Enter</span>
              </div>
            </div>
          </Workspace.Pane>

          <Workspace.Pane defaultSize={55} minSize={20}>
            <ResultTable
              result={result}
              title={tabs.active?.title ?? "query"}
              showSummary={false}
            />
          </Workspace.Pane>
        </Workspace.Split>
      </Workspace>
    </div>
  );
}
