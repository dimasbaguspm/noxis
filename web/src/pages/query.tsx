import { useEffect, useState } from "react";
import { Play } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { ResultTable } from "@/components/sql/result-table";
import { TableBrowser } from "@/components/sql/table-browser";
import { useApp } from "@/app/app-provider";
import { useQuery, useTableRows } from "@/hooks/use-data";
import { classify } from "@/lib/statements";

export default function QueryPage() {
  const { schema, active } = useApp();
  const [sql, setSql] = useState("SELECT 1");
  const { result: readResult, execute, loading } = useQuery(active);
  const { result: preview, label, pick } = useTableRows(active);

  // switching connection invalidates both grids
  useEffect(() => {
    setSql("SELECT 1");
  }, [active]);

  const kind = classify(sql);
  const result = preview ?? readResult;

  return (
    <div className="grid h-[calc(100svh-6rem)] gap-4 lg:grid-cols-[280px_1fr]">
      <Card className="min-h-0">
        <CardHeader>
          <CardTitle className="text-base">Tables</CardTitle>
          <CardDescription>{active ?? "no connection"}</CardDescription>
        </CardHeader>
        <CardContent className="min-h-0 flex-1">
          <TableBrowser tables={schema} onPick={(t) => void pick(t.schema, t.name)} />
        </CardContent>
      </Card>

      <div className="flex min-h-0 flex-col gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              Query
              {kind !== "empty" && <Badge variant={kind === "read" ? "secondary" : "destructive"}>{kind}</Badge>}
            </CardTitle>
            <CardDescription>
              reads run directly — writes belong on the DML / DDL pages where they get an approval
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            <Textarea
              value={sql}
              onChange={(e) => setSql(e.target.value)}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === "Enter") void execute(sql);
              }}
              className="min-h-32 font-mono"
              placeholder="SELECT …"
            />
            <div className="flex items-center gap-2">
              <Button onClick={() => void execute(sql)} disabled={loading || !active}>
                <Play /> Run
              </Button>
              <span className="text-xs text-muted-foreground">⌘/Ctrl + Enter</span>
            </div>
          </CardContent>
        </Card>

        <Card className="min-h-0 flex-1">
          <CardHeader>
            <CardTitle className="text-base">Result</CardTitle>
            <CardDescription>{label || "run a query or pick a table"}</CardDescription>
          </CardHeader>
          <CardContent className="min-h-0 overflow-auto">
            <ResultTable result={result} title={label || "query"} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
