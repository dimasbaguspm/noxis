import { Link } from "react-router-dom";
import { title } from "radash";
import { ExternalLink } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Change } from "@/lib/api";
import { PAGE_ROUTES } from "@/constants/routes";
import { P } from "@/components/ui/typography";

const STATUS_VARIANT: Record<
  string,
  "default" | "secondary" | "destructive" | "outline" | "success" | "warning"
> = {
  validated: "secondary",
  unverified: "warning",
  approved: "default",
  applied: "success",
  failed: "destructive",
  rejected: "outline",
};

// ChangeTable lists changes for triage. Actions live on the detail page, so the only thing here is "review" — the list stays a queue, not a workbench.
export function ChangeTable({ changes, emptyHint }: { changes: Change[]; emptyHint: string }) {
  if (changes.length === 0) {
    return <P size="sm">{emptyHint}</P>;
  }

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[90px]">{title("status")}</TableHead>
            <TableHead>{title("notes")}</TableHead>
            <TableHead className="w-[130px]">{title("instance")}</TableHead>
            <TableHead className="w-[220px]">{title("touches")}</TableHead>
            <TableHead className="w-[200px]">{title("impact")}</TableHead>
            <TableHead className="w-[110px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {changes.map((change) => {
            const id = change.id ?? "";
            const href = (
              change.kind === "ddl" ? PAGE_ROUTES.SQL.DDL_DETAIL : PAGE_ROUTES.SQL.DML_DETAIL
            )(id);
            return (
              <TableRow key={id} className="cursor-pointer">
                <TableCell>
                  <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>
                    {title(change.status ?? "")}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Link to={href} className="text-sm hover:underline">
                    {change.notes}
                  </Link>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{change.connection}</TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {(change.affectedTables ?? []).join(", ") || "—"}
                  {(change.affectedColumns ?? []).length > 0 && (
                    <span className="block text-muted-foreground/70">
                      {(change.affectedColumns ?? []).join(", ")}
                    </span>
                  )}
                </TableCell>
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
