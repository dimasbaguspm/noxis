import { Link } from "react-router-dom";
import { CircleHelp, Plus, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/app/auth-provider";
import { useConnections } from "@/hooks/use-connections";
import { useMongoChangesListQuery, useMongoConnections } from "@/hooks/use-mongodb";
import { FAMILY } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// MongoRequestsPage is the review queue: every write request waits here until someone approves and applies it. Reads never appear (they run in Explore).
export default function MongoRequestsPage() {
  const { canAccess } = useAuth();
  const { connections } = useMongoConnections();
  const { connections: mongoConnections } = useConnections(FAMILY.MONGODB);
  const { changes, isFetching: loading, refetch: reload } = useMongoChangesListQuery();
  const mayWrite = mongoConnections.some((c) => canAccess(c.name, "write"));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-sm text-muted-foreground">
          {connections.length === 0
            ? "register a mongodb server from the dashboard first"
            : "writes are validated on a scratch collection, then reviewed here"}
        </span>
        <span className="flex-1" />
        <Button variant="outline" size="sm" onClick={() => void reload()} disabled={loading}>
          <RefreshCw className={cn(loading && "animate-spin")} /> Refresh
        </Button>
        {mayWrite && (
          <Button size="sm" render={<Link to={PAGE_ROUTES.MONGODB.REQUEST_NEW} />}>
            <Plus /> New request
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Requests</CardTitle>
          <CardDescription>
            {changes.length} request(s) · approved ones can be applied, rejected ones stay as artifacts
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Notes</TableHead>
                <TableHead className="w-[120px]">Status</TableHead>
                <TableHead>Connection</TableHead>
                <TableHead>Target</TableHead>
                <TableHead className="w-[120px]">Estimated</TableHead>
                <TableHead className="w-[110px]">Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {changes.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-muted-foreground">
                    nothing queued
                  </TableCell>
                </TableRow>
              )}
              {changes.map((change) => (
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
                  <TableCell>
                    <Button size="xs" variant="outline" render={<Link to={PAGE_ROUTES.MONGODB.REQUEST_DETAIL(change.id ?? "")} />}>
                      Review
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <div className="flex items-center gap-2 text-xs text-muted-foreground">
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
