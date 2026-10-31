import { Link, useParams } from "react-router-dom";
import { Inbox, Search as SearchIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useSearchChangesQuery } from "@/hooks/use-search-changes";
import { PAGE_ROUTES } from "@/constants/routes";
import { compact } from "@/lib/format";

// STATUS is the badge tone per change status, so a queue reads at a glance.
const STATUS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "default",
  unverified: "outline",
  approved: "secondary",
  applied: "secondary",
  applying: "outline",
  failed: "destructive",
  rejected: "outline",
};

// SearchRequestsPage is the search capability's own review queue: every change a search platform raised, with the
// impact the platform counted, so a reviewer approves a number rather than a query.
export default function SearchRequestsPage() {
  const { driver = "" } = useParams();
  const { connection } = useDriverConnections();
  const { changes, total, isPending } = useSearchChangesQuery(driver, connection);

  if (isPending) return <p className="text-sm text-muted-foreground">Loading…</p>;

  if (changes.length === 0) {
    return (
      <Empty
        icon={Inbox}
        title="No search changes"
        description="A delete raised from the search page is reviewed here, with the impact the platform reported."
      />
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto">
      <p className="text-xs text-muted-foreground">
        {compact(total)} change{total === 1 ? "" : "s"} for this driver
      </p>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Notes</TableHead>
            <TableHead>Target</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Impact</TableHead>
            <TableHead>Created</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {changes.map((change) => (
            <TableRow key={change.id}>
              <TableCell>
                <Link
                  to={PAGE_ROUTES.SEARCH_REQUEST_DETAIL(driver, change.id ?? "")}
                  className="font-medium underline-offset-4 hover:underline"
                >
                  {change.notes || change.id?.slice(0, 8)}
                </Link>
              </TableCell>
              <TableCell className="font-mono text-xs">
                {change.affectedTables?.[0] ?? "—"}
              </TableCell>
              <TableCell>
                <Badge variant={STATUS[change.status ?? ""] ?? "outline"}>
                  {change.status ?? "unknown"}
                </Badge>
              </TableCell>
              <TableCell className="text-xs">
                {change.estAffected ? `${compact(change.estAffected)} event(s)` : "—"}
              </TableCell>
              <TableCell className="text-xs text-muted-foreground">
                {change.createdAt ? new Date(change.createdAt * 1000).toLocaleString() : "—"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="flex items-center gap-1 text-xs text-muted-foreground">
        <SearchIcon className="size-3" /> raised from the search page
      </p>
    </div>
  );
}
