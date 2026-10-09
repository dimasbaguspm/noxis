import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlTabs } from "@/components/sql/sql-tabs";
import { TableBrowser } from "@/components/sql/table-browser";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { useQuery } from "@/hooks/use-data";
import { useSqlTabs } from "@/hooks/use-sql-tabs";
import { api, type Result } from "@/lib/api";
import { classify } from "@/lib/statements";
import { Play } from "lucide-react";
import { useState } from "react";

export default function QueryPage() {
  const { can, isAdmin } = useApp();
  const { connection, connections, busy, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const tabs = useSqlTabs();
  const [analyzed, setAnalyzed] = useState<Result | null>(null);
  const { result: readResult, execute, loading } = useQuery(connection);

  const analyze = useAction(async (table: { schema: string; name: string }) => {
    if (!connection) return;
    setAnalyzed(await api.ops.analyze(connection, `${table.schema}.${table.name}`));
  });

  const sql = tabs.active?.sql ?? "";
  const kind = classify(sql);
  const result = analyzed ?? readResult;

  // No SQL connection to query: explain it and offer the way in, instead of
  // rendering empty panes.
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
        {!busy && isAdmin && (
          <CardContent>
            <ConnectionDialog trigger="New connection" wizard />
          </CardContent>
        )}
      </Card>
    );
  }

  return (
    <div className="flex h-[calc(100svh-7rem)] flex-col">
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 min-w-0 flex-1">
        <ResizablePanel defaultSize={16} minSize={12}>
          <Card className="mr-2 h-full min-h-0 min-w-0">
            <CardContent className="flex min-h-0 min-w-0 flex-col gap-2">
              <ConnectionSelect value={connection} onChange={pick} className="w-full" />
              <div className="min-h-0 flex-1">
                <TableBrowser
                  tables={tables}
                  onRecords={(t) => tabs.add(`SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`)}
                  onAnalyze={can(connection, "write") ? (t) => void analyze.run(t) : undefined}
                />
              </div>
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={84} minSize={50} className="min-w-0 overflow-hidden">
          <ResizablePanelGroup orientation="vertical" className="flex min-w-0 flex-col gap-2 pl-2">
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
                  />
                  <div className="flex items-center gap-2">
                    <Button onClick={() => void execute(sql)} disabled={loading || !connection}>
                      <Play /> Run
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => void execute(`EXPLAIN ${sql}`)}
                      disabled={kind !== "read" || !connection}
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
                <CardHeader className="shrink-0">
                  <CardTitle className="text-base">Result</CardTitle>
                </CardHeader>
                <CardContent className="flex min-h-0 min-w-0 flex-1 flex-col">
                  <ResultTable result={result} title={tabs.active?.title ?? "query"} />
                </CardContent>
              </Card>
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
