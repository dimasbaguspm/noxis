import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { downloadCSV, toCSV } from "@/lib/csv";
import type { Result } from "@/lib/api";
import { P } from "@/components/ui/typography";

// ResultSummary reports what a run returned: message, duration, and the affected or returned counts.
export function ResultSummary({ result }: { result: Result | null }) {
  if (!result) return null;
  return (
    <P className="flex min-w-0 flex-wrap items-center gap-x-3">
      <span className="break-words">{result.message}</span>
      <span>Duration: {result.durationMs ?? 0} ms</span>
      {(result.rowsReturned ?? 0) > 0 && <span>Returned: {result.rowsReturned}</span>}
      {(result.affected ?? 0) > 0 && <span>Affected: {result.affected}</span>}
      {(result.estimatedRows ?? 0) > 0 && <span>Plan rows: ~{result.estimatedRows}</span>}
    </P>
  );
}

// ResultTable renders a query grid and can export it as CSV; the summary moves to the page footer when the page asks for it.
export function ResultTable({
  result,
  title = "result",
  showSummary = true,
}: {
  result: Result | null;
  title?: string;
  showSummary?: boolean;
}) {
  if (!result) {
    return (
      <Empty title="Nothing to show yet" description="Run an action to see its result here." />
    );
  }

  const columns = result.columns ?? [];
  const rows = result.rows ?? [];

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">
      <div className="flex min-w-0 items-center gap-2">
        {showSummary && <ResultSummary result={result} />}
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
        // the grid owns its scrolling, so the header has a scroll port to stick to
        <div className="min-h-0 flex-1 overflow-hidden rounded-md border border-border">
          <Table containerClassName="h-full overflow-auto">
            <TableHeader className="sticky top-0 z-10 bg-card">
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
                    No rows
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
