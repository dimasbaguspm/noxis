import { useState } from "react";
import { Download, Maximize2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
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

// ResultGrid is the table itself: one scroll port, a sticky header, and no opinion about the box it sits in.
function ResultGrid({ columns, rows }: { columns: string[]; rows: string[][] }) {
  return (
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
  );
}

// ResultTable renders a query grid, exports it as CSV, and can open it over the whole window; the summary moves to the
// page footer when the page asks for it.
export function ResultTable({
  result,
  title = "result",
  showSummary = true,
}: {
  result: Result | null;
  title?: string;
  showSummary?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  if (!result) {
    return (
      <Empty title="Nothing to show yet" description="Run an action to see its result here." />
    );
  }

  const columns = result.columns ?? [];
  const rows = result.rows ?? [];
  const csv = () => downloadCSV(`${title}.csv`, toCSV(columns, rows));

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden">
      <div className="flex min-w-0 items-center gap-2">
        {showSummary && <ResultSummary result={result} />}
        <span className="flex-1" />
        {columns.length > 0 && (
          <>
            <Button size="sm" variant="outline" onClick={csv}>
              <Download /> CSV
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setExpanded(true)}
              aria-label="Full screen"
              title="Full screen"
            >
              <Maximize2 />
            </Button>
          </>
        )}
      </div>

      {columns.length > 0 && <ResultGrid columns={columns} rows={rows} />}

      <Dialog open={expanded} onOpenChange={setExpanded}>
        <DialogContent className="flex h-[92svh] w-[96vw] max-w-none flex-col sm:max-w-none">
          <DialogHeader className="flex-row items-center gap-2">
            <DialogTitle className="truncate">{title}</DialogTitle>
            <span className="flex-1" />
            <Button size="sm" variant="outline" onClick={csv}>
              <Download /> CSV
            </Button>
          </DialogHeader>
          <ResultGrid columns={columns} rows={rows} />
        </DialogContent>
      </Dialog>
    </div>
  );
}
