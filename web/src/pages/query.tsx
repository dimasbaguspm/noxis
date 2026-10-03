import { useEffect, useState } from "react";
import { Activity, Play } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Textarea } from "@/components/ui/textarea";
import { ResultTable } from "@/components/sql/result-table";
import { TableBrowser } from "@/components/sql/table-browser";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { useQuery, useTableRows } from "@/hooks/use-data";
import { api, type Result } from "@/lib/api";
import { classify } from "@/lib/statements";

export default function QueryPage() {
  const { schema, active } = useApp();
  const [sql, setSql] = useState("SELECT 1");
  const [analyzed, setAnalyzed] = useState<Result | null>(null);
  const { result: readResult, execute, loading } = useQuery(active);
  const { result: preview, label, pick } = useTableRows(active);

  useEffect(() => {
    setSql("SELECT 1");
    setAnalyzed(null);
  }, [active]);

  const analyze = useAction(async (table: { schema: string; name: string }) => {
    if (!active) return;
    setAnalyzed(await api.ops.analytics(active, `${table.schema}.${table.name}`));
  });

  const kind = classify(sql);
  const result = analyzed ?? preview ?? readResult;

  return (
    <ResizablePanelGroup orientation="horizontal" className="h-[calc(100svh-7rem)]">
      <ResizablePanel defaultSize={24} minSize={16}>
        <Card className="mr-2 h-full min-h-0">
          <CardHeader>
            <CardTitle className="text-base">Tables</CardTitle>
            <CardDescription>{active ?? "no connection"}</CardDescription>
          </CardHeader>
          <CardContent className="min-h-0 flex-1">
            <TableBrowser
              tables={schema}
              onPick={(t) => void pick(t.schema, t.name)}
              onRecords={(t) => void pick(t.schema, t.name)}
              onAnalyze={(t) => void analyze.run(t)}
            />
          </CardContent>
        </Card>
      </ResizablePanel>

      <ResizableHandle withHandle />

      <ResizablePanel defaultSize={76}>
        <ResizablePanelGroup orientation="vertical" className="ml-2">
          <ResizablePanel defaultSize={45} minSize={20}>
            <Card className="h-full">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  Query
                  {kind !== "empty" && <Badge variant={kind === "read" ? "secondary" : "destructive"}>{kind}</Badge>}
                </CardTitle>
                <CardDescription>
                  reads run directly — writes go through the DML / DDL pages for approval
                </CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-2">
                <Textarea
                  value={sql}
                  onChange={(e) => setSql(e.target.value)}
                  onKeyDown={(e) => {
                    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") void execute(sql);
                  }}
                  className="min-h-24 font-mono"
                  placeholder="SELECT …"
                />
                <div className="flex items-center gap-2">
                  <Button onClick={() => void execute(sql)} disabled={loading || !active}>
                    <Play /> Run
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => {
                      if (kind === "read") void execute(sql);
                    }}
                    disabled={kind !== "read" || !active}
                  >
                    <Activity /> Explain
                  </Button>
                  <span className="text-xs text-muted-foreground">⌘/Ctrl + Enter</span>
                </div>
              </CardContent>
            </Card>
          </ResizablePanel>

          <ResizableHandle withHandle />

          <ResizablePanel defaultSize={55} minSize={20}>
            <Card className="mt-2 h-full min-h-0">
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
  );
}
