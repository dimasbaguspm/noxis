import { Download } from "lucide-react";
import { title as startCase } from "radash";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { downloadCSV, toCSV } from "@/lib/csv";
import type { Result } from "@/lib/api";
import { P } from "@/components/ui/typography";

// ResultTable renders a query grid and can export it as CSV.
export function ResultTable({ result, title = "result" }: { result: Result | null; title?: string }) {
  if (!result) {
    return <Empty title="Nothing to show yet" description="Run an action to see its result here." />;
  }

  const columns = result.columns ?? [];
  const rows = result.rows ?? [];

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">
      <div className="flex min-w-0 items-center gap-2">
        <P className="flex min-w-0 flex-wrap items-center gap-x-3">
          <span className="break-words">{result.message}</span>
          <span>{startCase("duration")}: {result.durationMs ?? 0} ms</span>
          {(result.rowsReturned ?? 0) > 0 && <span>{startCase("returned")}: {result.rowsReturned}</span>}
          {(result.affected ?? 0) > 0 && <span>{startCase("affected")}: {result.affected}</span>}
          {(result.estimatedRows ?? 0) > 0 && <span>{startCase("plan rows")}: ~{result.estimatedRows}</span>}
        </P>
        <span className="flex-1" />
        {columns.length > 0 && (
          <Button
            size="sm"
            variant="outline"
            onClick={() => downloadCSV(`${title}.csv`, toCSV(columns, rows))}
          >
            <Download /> CSV
          </Button>
        )}
      </div>

      {columns.length > 0 && (
        <div className="min-h-0 flex-1 overflow-auto rounded-md border border-border">
          <Table>
            <TableHeader className="sticky top-0 bg-card">
              <TableRow>
                {columns.map((c) => (
                  <TableHead key={c}>{c}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.join("\u0000")}>
                  {row.map((cell, j) => (
                    <TableCell key={columns[j]} className="font-mono text-xs">
                      {cell === "NULL" ? <span className="text-muted-foreground">NULL</span> : cell}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={columns.length} className="text-muted-foreground">
                    {startCase("no rows")}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
