import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlTabs } from "@/components/sql/sql-tabs";
import { TableBrowser } from "@/components/sql/table-browser";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Workspace } from "@/components/layout/workspace";
import { useSearchParams } from "react-router-dom";
import { useAuth } from "@/app/auth-provider";
import { useConnectionSelection } from "@/hooks/use-connection";
import { useAnalyzeMutation, useConnectionSchemaQuery, useRunQueryMutation } from "@/hooks/use-sql";
import { useSqlTabs } from "@/hooks/use-sql-tabs";
import { classify } from "@/lib/statements";
import { Play } from "lucide-react";

export default function QueryPage() {
  const { canAccess, isSuper } = useAuth();
  const [params] = useSearchParams();
  const { connection, connections, busy, pick } = useConnectionSelection(
    params.get("connection") ?? undefined,
    params.get("instance") ?? undefined,
  );

  const tabs = useSqlTabs(params.get("sql") ?? undefined);
  const tables = useConnectionSchemaQuery(connection).data ?? [];
  const run = useRunQueryMutation();
  const readResult = run.data;
  const loading = run.isPending;
  const execute = (sql: string) => {
    if (!connection || !sql.trim()) return;
    void run.mutateAsync({ connection, sql });
  };

  const analyze = useAnalyzeMutation();
  const analyzed = analyze.data;
  const onAnalyze = (table: { schema: string; name: string }) =>
    connection ? analyze.mutate({ connection, table: `${table.schema}.${table.name}` }) : undefined;

  const sql = tabs.active?.sql ?? "";
  const canRun = !!connection && sql.trim().length > 0;
  const kind = classify(sql);
  const result = analyzed ?? readResult ?? null;

  // No SQL connection to query: explain it and offer the way in, instead of rendering empty panes.
  if (connections.length === 0) {
    return (
      <Card className="mx-auto mt-8 max-w-lg">
        <CardHeader>
          <CardTitle>{busy ? "Loading connections…" : "No connection found"}</CardTitle>
          <CardDescription>
            {busy
              ? "Fetching your registered SQL databases."
              : "Register a SQL database to start browsing and querying."}
          </CardDescription>
        </CardHeader>
        {!busy && isSuper && (
          <CardContent>
            <ConnectionDialog trigger="New connection" wizard />
          </CardContent>
        )}
      </Card>
    );
  }

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        leftLabel="Query filters"
        rightLabel="Query editor and result"
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-2">
            <div className="grid gap-1.5">
              <Label>Instance</Label>
              <ConnectionSelect value={connection} onChange={pick} className="w-full" />
            </div>
            <div className="min-h-0 flex-1">
              <TableBrowser
                tables={tables}
                onRecords={(t) => tabs.add(`SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`)}
                onAnalyze={canAccess(connection, "write") ? onAnalyze : undefined}
              />
            </div>
          </div>
        }
        right={
          <ResizablePanelGroup orientation="vertical" className="flex min-h-0 flex-1 flex-col gap-2">
            <ResizablePanel defaultSize={45} minSize={20}>
              <Card className="flex h-full min-w-0 flex-col">
                <CardContent className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">
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
                      <Play /> {loading ? "Running…" : "Run"}
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
                </CardContent>
              </Card>
            </ResizablePanel>

            <ResizableHandle withHandle />

            <ResizablePanel defaultSize={55} minSize={20}>
              <Card className="flex h-full min-h-0 min-w-0 flex-col">
                <CardContent className="flex min-h-0 min-w-0 flex-1 flex-col">
                  <ResultTable result={result} title={tabs.active?.title ?? "query"} />
                </CardContent>
              </Card>
            </ResizablePanel>
          </ResizablePanelGroup>
        }
      />
    </div>
  );
}
