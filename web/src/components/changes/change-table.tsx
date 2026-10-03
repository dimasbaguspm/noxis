import { Link } from "react-router-dom";
import { Check, ExternalLink, Play } from "lucide-react";
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

// ChangeTable lists changes; review happens on the detail page, and rows are
// artifacts (never deleted).
export function ChangeTable({
  changes,
  onApprove,
  onApply,
  emptyHint,
}: {
  changes: Change[];
  onApprove: (id: string) => Promise<void>;
  onApply: (id: string) => Promise<void>;
  emptyHint: string;
}) {
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
            <TableHead className="w-[130px]">connection</TableHead>
            <TableHead className="w-[200px]">estimates</TableHead>
            <TableHead className="w-[190px]">actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {changes.map((change) => {
            const id = change.id ?? "";
            return (
              <TableRow key={id}>
                <TableCell>
                  <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
                </TableCell>
                <TableCell>
                  <Link to={`/${change.kind ?? "dml"}/${id}`} className="text-sm hover:underline">
                    {change.notes}
                  </Link>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{change.connection}</TableCell>
                <TableCell className="text-xs text-muted-foreground tabular-nums">
                  {change.status === "applied"
                    ? `${change.appliedDurationMs ?? 0} ms · ${change.appliedAffected ?? 0} rows`
                    : `${change.estDurationMs ?? 0} ms · ~${change.estAffected ?? 0} rows`}
                </TableCell>
                <TableCell className="flex flex-wrap gap-1.5">
                  {(change.status === "validated" || change.status === "unverified") && (
                    <Button size="xs" variant="secondary" onClick={() => void onApprove(id)}>
                      <Check /> Approve
                    </Button>
                  )}
                  {change.status === "approved" && (
                    <Button size="xs" onClick={() => void onApply(id)}>
                      <Play /> Apply
                    </Button>
                  )}
                  <Button size="xs" variant="ghost" render={<Link to={`/${change.kind ?? "dml"}/${id}`} />}>
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
