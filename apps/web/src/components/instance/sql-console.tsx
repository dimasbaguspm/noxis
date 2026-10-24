import { useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, Play } from "lucide-react";
import { ResultTable } from "@/components/sql/result-table";
import { SqlEditor } from "@/components/sql/sql-editor";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useConnectionSchemaQuery, useRunQueryMutation } from "@/hooks/use-sql";
import { PAGE_ROUTES } from "@/lib/routes";

// SqlConsole is a one-statement read console on the instance page; tabs, writes and history stay on the full query page.
export function SqlConsole({ connection }: { connection: string }) {
  const [sql, setSql] = useState("");
  const tables = useConnectionSchemaQuery(connection).data ?? [];
  const run = useRunQueryMutation();
  const canRun = sql.trim().length > 0 && !run.isPending;

  function execute() {
    if (!connection || !canRun) return;
    run.mutate({ connection, sql });
  }

  return (
    <Card>
      <CardHeader className="gap-1">
        <CardTitle className="text-base">Query</CardTitle>
        <CardDescription>
          Read-only SQL against {connection}. Writes go through the change flow on the full query page.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <SqlEditor
          value={sql}
          onChange={setSql}
          tables={tables}
          height="12rem"
          placeholder="SELECT …"
          onRun={execute}
        />
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" onClick={execute} disabled={!canRun}>
            <Play /> Run
          </Button>
          <Button
            size="sm"
            variant="outline"
            render={<Link to={`${PAGE_ROUTES.SQL.QUERY}?connection=${encodeURIComponent(connection)}`} />}
          >
            <ExternalLink /> Full query page
          </Button>
          {run.error instanceof Error && (
            <span className="text-xs text-destructive">{run.error.message}</span>
          )}
        </div>
        {run.data && (
          <div className="h-72">
            <ResultTable result={run.data} title={connection} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
