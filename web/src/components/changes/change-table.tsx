import { Link } from "react-router-dom";
import { ExternalLink } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Change } from "@/lib/api";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applied: "default",
  failed: "destructive",
  rejected: "outline",
};

// ChangeTable lists changes for triage. Actions live on the detail page, so
// the only thing here is "review" — the list stays a queue, not a workbench.
export function ChangeTable({ changes, emptyHint }: { changes: Change[]; emptyHint: string }) {
  if (changes.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyHint}</p>;
  }

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[90px]">status</TableHead>
            <TableHead>notes</TableHead>
            <TableHead className="w-[130px]">instance</TableHead>
            <TableHead className="w-[200px]">impact</TableHead>
            <TableHead className="w-[110px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {changes.map((change) => {
            const id = change.id ?? "";
            const href = `/${change.kind ?? "dml"}/${id}`;
            return (
              <TableRow key={id} className="cursor-pointer">
                <TableCell>
                  <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
                </TableCell>
                <TableCell>
                  <Link to={href} className="text-sm hover:underline">
                    {change.notes}
                  </Link>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{change.connection}</TableCell>
                <TableCell className="text-xs text-muted-foreground tabular-nums">
                  {change.status === "applied"
                    ? `${change.appliedDurationMs ?? 0} ms · ${change.appliedAffected ?? 0} rows`
                    : `${change.estDurationMs ?? 0} ms · ~${change.estAffected ?? 0} rows`}
                </TableCell>
                <TableCell>
                  <Button size="xs" variant="ghost" render={<Link to={href} />}>
                    <ExternalLink /> Review
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
