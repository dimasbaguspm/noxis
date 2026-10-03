import { ConnectionSelect } from "@/components/connection/connection-select";
import { ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlTabs } from "@/components/sql/sql-tabs";
import { TableBrowser } from "@/components/sql/table-browser";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAction } from "@/hooks/use-action";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { useQuery } from "@/hooks/use-data";
import { useSqlTabs } from "@/hooks/use-sql-tabs";
import { api, type Result } from "@/lib/api";
import { classify } from "@/lib/statements";
import { CircleHelp, Play } from "lucide-react";
import { useState } from "react";

export default function QueryPage() {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const tabs = useSqlTabs();
  const [analyzed, setAnalyzed] = useState<Result | null>(null);
  const { result: readResult, execute, loading } = useQuery(connection);

  const analyze = useAction(async (table: { schema: string; name: string }) => {
    if (!connection) return;
    setAnalyzed(await api.ops.analytics(connection, `${table.schema}.${table.name}`));
  });

  const sql = tabs.active?.sql ?? "";
  const kind = classify(sql);
  const result = analyzed ?? readResult;

  return (
    <div className="flex h-[calc(100svh-7rem)] flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <ConnectionSelect value={connection} onChange={pick} />
        <Badge variant={kind === "read" ? "secondary" : "destructive"}>{kind}</Badge>
        <Tooltip>
          <TooltipTrigger render={<span className="text-muted-foreground" />}>
            <CircleHelp className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent>
            Reads run directly. Writes belong on the DML or DDL page, where they get validated and approved.
          </TooltipContent>
        </Tooltip>
      </div>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={24} minSize={16}>
          <Card className="mr-2 h-full min-h-0">
            <CardContent className="min-h-0 flex-1">
              <TableBrowser
                tables={tables}
                onRecords={(t) => tabs.add(`SELECT * FROM ${t.schema}.${t.name} LIMIT 100;`)}
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
                <CardContent className="flex min-h-0 flex-1 flex-col gap-2 pt-4">
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
              <Card className="mt-2 flex h-full min-h-0 flex-col">
                <CardHeader>
                  <CardTitle className="text-base">Result</CardTitle>
                </CardHeader>
                <CardContent className="min-h-0 overflow-auto">
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
