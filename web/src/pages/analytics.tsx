import { useState } from "react";
import { Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ResultTable } from "@/components/sql/result-table";
import { TableBrowser } from "@/components/sql/table-browser";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { api, type Result } from "@/lib/api";

// AnalyticsPage refreshes planner statistics for a table.
export default function AnalyticsPage() {
  const { schema, active } = useApp();
  const [table, setTable] = useState<string>();
  const [result, setResult] = useState<Result | null>(null);

  const analyze = useAction(async () => {
    if (!active || !table) return;
    setResult(await api.ops.analytics(active, table));
  });

  return (
    <div className="grid h-[calc(100svh-6rem)] gap-4 lg:grid-cols-[280px_1fr]">
      <Card className="min-h-0">
        <CardHeader>
          <CardTitle className="text-base">Tables</CardTitle>
          <CardDescription>{active ?? "no connection"}</CardDescription>
        </CardHeader>
        <CardContent className="min-h-0 flex-1">
          <TableBrowser tables={schema} onPick={(t) => setTable(`${t.schema}.${t.name}`)} />
        </CardContent>
      </Card>

      <div className="flex min-h-0 flex-col gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Analytics</CardTitle>
            <CardDescription>ANALYZE / ANALYZE TABLE — refresh planner statistics</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap items-center gap-2">
            <Select value={table} onValueChange={(value) => setTable(String(value))}>
              <SelectTrigger className="w-72">
                <SelectValue placeholder="pick a table" />
              </SelectTrigger>
              <SelectContent>
                {schema.map((t) => (
                  <SelectItem key={`${t.schema}.${t.name}`} value={`${t.schema}.${t.name}`}>
                    {t.schema}.{t.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button onClick={() => void analyze.run()} disabled={analyze.loading || !table}>
              <Activity /> Analyze
            </Button>
          </CardContent>
        </Card>

        <Card className="min-h-0 flex-1">
          <CardHeader>
            <CardTitle className="text-base">Result</CardTitle>
            <CardDescription>{table ?? "no table selected"}</CardDescription>
          </CardHeader>
          <CardContent>
            <ResultTable result={result} title={`analyze-${table ?? "table"}`} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
