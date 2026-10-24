import { useState } from "react";
import { Link } from "react-router-dom";
import { CircleHelp, Inbox, Plus, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/app/auth-provider";
import { useConnections } from "@/hooks/use-connections";
import { useMongoChangesListQuery } from "@/hooks/use-mongodb";
import { FAMILY } from "@/lib/families";
import { Workspace } from "@/components/layout/workspace";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// MongoRequestsPage is the review queue: every write request waits here until someone approves and applies it. Reads never appear (they run in Explore).
export default function MongoRequestsPage() {
  const { canAccess } = useAuth();
  const { connections: mongoConnections } = useConnections(FAMILY.MONGODB);
  const { changes, isFetching: loading, refetch: reload } = useMongoChangesListQuery();
  const [status, setStatus] = useState("all");
  const mayWrite = mongoConnections.some((c) => canAccess(c.name, "write"));
  const visible = status === "all" ? changes : changes.filter((change) => change.status === status);

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <Button variant="outline" size="sm" onClick={() => void reload()} disabled={loading}>
              <RefreshCw className={cn(loading && "animate-spin")} /> Refresh
            </Button>
            {mayWrite && (
              <Button size="sm" render={<Link to={PAGE_ROUTES.MONGODB.REQUEST_NEW} />}>
                <Plus /> New request
              </Button>
            )}
            <div className="flex flex-col gap-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(value) => setStatus(String(value))}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="validated">Validated</SelectItem>
                  <SelectItem value="approved">Approved</SelectItem>
                  <SelectItem value="applied">Applied</SelectItem>
                  <SelectItem value="rejected">Rejected</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="mt-auto flex items-start gap-2 text-xs text-muted-foreground">
              <Tooltip>
                <TooltipTrigger render={<span />}>
                  <CircleHelp className="size-3.5" />
                </TooltipTrigger>
                <TooltipContent>
                  A document change cannot be rolled back, so the review page shows the dry run and the trail instead of
                  a rollback script.
                </TooltipContent>
              </Tooltip>
              no rollback exists for document changes
            </div>
          </div>
        }
        right={
          loading ? (
            <div className="flex flex-col gap-2">
              {[1, 2, 3, 4, 5].map((row) => (
                <Skeleton key={row} className="h-9 w-full" />
              ))}
            </div>
          ) : visible.length === 0 ? (
            <Empty icon={Inbox} title="No requests" description="MongoDB write requests waiting for review appear here." />
          ) : (
            <div className="min-h-0 flex-1 overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Notes</TableHead>
                    <TableHead className="w-[120px]">Status</TableHead>
                    <TableHead>Connection</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead className="w-[120px]">Estimated</TableHead>
                    <TableHead className="w-[110px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((change) => (
                    <TableRow key={change.id}>
                      <TableCell className="text-sm">{change.notes}</TableCell>
                      <TableCell>
                        <Badge variant={statusVariant(change.status)}>{change.status}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{change.connection}</TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {(change.affectedTables ?? []).join(", ")}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {change.estAffected ?? 0} doc(s)
                      </TableCell>
                      <TableCell className="text-right">
                        <Button size="xs" variant="outline" render={<Link to={PAGE_ROUTES.MONGODB.REQUEST_DETAIL(change.id ?? "")} />}>
                          Review
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )
        }
      />
    </div>
  );
}

function statusVariant(status?: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "validated":
      return "secondary";
    case "approved":
    case "applied":
      return "default";
    case "failed":
      return "destructive";
  }
  return "outline";
}
