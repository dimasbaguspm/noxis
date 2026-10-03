import { useEffect, useState } from "react";
import { Activity, Play } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { TableBrowser } from "@/components/sql/table-browser";
import { useAction } from "@/hooks/use-action";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { useQuery, useTableRows } from "@/hooks/use-data";
import { api, type Result } from "@/lib/api";
import { classify } from "@/lib/statements";

export default function QueryPage() {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const [sql, setSql] = useState("SELECT 1");
  const [analyzed, setAnalyzed] = useState<Result | null>(null);
  const { result: readResult, execute, loading } = useQuery(connection);
  const { result: preview, label, pick: pickTable } = useTableRows(connection);

  useEffect(() => {
    setAnalyzed(null);
  }, [connection]);

  const analyze = useAction(async (table: { schema: string; name: string }) => {
    if (!connection) return;
    setAnalyzed(await api.ops.analytics(connection, `${table.schema}.${table.name}`));
  });

  const kind = classify(sql);
  const result = analyzed ?? preview ?? readResult;

  return (
    <div className="flex h-[calc(100svh-7rem)] flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <ConnectionSelect value={connection} onChange={pick} />
        <Badge variant={kind === "read" ? "secondary" : "destructive"}>{kind}</Badge>
        <span className="text-xs text-muted-foreground">
          reads run directly — writes go through DML / DDL for approval
        </span>
      </div>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={24} minSize={16}>
          <Card className="mr-2 h-full min-h-0">
            <CardHeader>
              <CardTitle className="text-base">Tables</CardTitle>
              <CardDescription>{connection ?? "no instance"}</CardDescription>
            </CardHeader>
            <CardContent className="min-h-0 flex-1">
              <TableBrowser
                tables={tables}
                onPick={(t) => setSql(`SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`)}
                onRecords={(t) => void pickTable(t.schema, t.name)}
                onAnalyze={(t) => void analyze.run(t)}
              />
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={76}>
          <ResizablePanelGroup orientation="vertical" className="ml-2">
            <ResizablePanel defaultSize={45} minSize={20}>
              <Card className="flex h-full flex-col">
                <CardHeader>
                  <CardTitle className="text-base">SQL</CardTitle>
                  <CardDescription>⌘/Ctrl + Enter runs · completion knows this instance's schema</CardDescription>
                </CardHeader>
                <CardContent className="flex min-h-0 flex-1 flex-col gap-2">
                  <SqlEditor
                    value={sql}
                    onChange={setSql}
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
                      <Activity /> Explain
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </ResizablePanel>

            <ResizableHandle withHandle />

            <ResizablePanel defaultSize={55} minSize={20}>
              <Card className="mt-2 flex h-full min-h-0 flex-col">
                <CardHeader>
                  <CardTitle className="text-base">Result</CardTitle>
                  <CardDescription>{label || "run a query or pick a table"}</CardDescription>
                </CardHeader>
                <CardContent className="min-h-0 overflow-auto">
                  <ResultTable result={result} title={label || "query"} />
                </CardContent>
              </Card>
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
