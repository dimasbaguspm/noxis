import { Link, useParams } from "react-router-dom";
import { CircleHelp, Inbox, Plus, RefreshCw, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Pagination } from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/app/auth-provider";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useMongoChangesListQuery } from "@/hooks/use-mongodb";
import { useChangeListUrlParams } from "@/hooks/use-changes";
import { InstanceFilter } from "@/components/filters/instance-filter";
import { QueuePanel } from "@/components/layout/queue-panel";
import { Workspace } from "@/components/layout/workspace";
import { PAGE_ROUTES } from "@/constants/routes";
import { cn } from "@/lib/utils";

const PER_PAGE = 20;
const STATUS_OPTIONS = ["validated", "approved", "applied", "rejected", "failed"];
const SKELETON_ROWS = [1, 2, 3, 4, 5];

// MongoRequestsPage is the review queue: every write request waits here until someone approves and applies it. Reads never
// appear (they run in Explore). The filters and the action row match the change queues, so a reviewer learns one layout.
export default function MongoRequestsPage() {
  const { driver = "" } = useParams();
  const { canAccess } = useAuth();
  const { connections: mongoConnections } = useDriverConnections();
  const { connections, name, status, page, setConnections, setName, setStatus, setPage } =
    useChangeListUrlParams();
  const { changes, total = 0, isPending, isFetching, refetch } = useMongoChangesListQuery();
  const mayWrite = mongoConnections.some((c) => canAccess(c.name, "write"));
  const narrowed = connections.length > 0 || name.trim() !== "";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <QueuePanel
            actions={
              <>
                {mayWrite && (
                  <Button size="sm" render={<Link to={PAGE_ROUTES.REQUEST_NEW(driver)} />}>
                    <Plus /> New request
                  </Button>
                )}
                <Button variant="outline" onClick={() => void refetch()} disabled={isFetching}>
                  <RefreshCw className={cn(isFetching && "animate-spin")} /> Refresh
                </Button>
              </>
            }
          >
            <InstanceFilter
              options={mongoConnections.map((c) => c.name)}
              values={connections}
              onChange={setConnections}
            />
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="request-name">Name</Label>
              <Input
                id="request-name"
                placeholder="notes"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(value) => setStatus(String(value))}>
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {(value) =>
                      value === "all"
                        ? "All statuses"
                        : `${String(value).charAt(0).toUpperCase()}${String(value).slice(1)}`
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  {STATUS_OPTIONS.map((s) => (
                    <SelectItem key={s} value={s}>
                      {s[0]!.toUpperCase() + s.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button variant="outline" onClick={() => void refetch()} disabled={isFetching}>
              <Search /> Search
            </Button>
            <div className="flex items-start gap-2 text-xs text-muted-foreground">
              <Tooltip>
                <TooltipTrigger render={<span />}>
                  <CircleHelp className="size-3.5" />
                </TooltipTrigger>
                <TooltipContent>
                  A document change cannot be rolled back, so the review page shows the dry run and
                  the trail instead of a rollback script.
                </TooltipContent>
              </Tooltip>
              No rollback exists for document changes
            </div>
          </QueuePanel>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto">
            {isPending ? (
              <div className="flex flex-col gap-2">
                {SKELETON_ROWS.map((row) => (
                  <Skeleton key={row} className="h-9 w-full" />
                ))}
              </div>
            ) : changes.length === 0 ? (
              <Empty
                className="flex-1"
                icon={Inbox}
                title="No requests"
                description={
                  narrowed
                    ? "No request matches these filters"
                    : "MongoDB write requests waiting for review appear here."
                }
              />
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
                    {changes.map((change) => (
                      <TableRow key={change.id}>
                        <TableCell className="text-sm">{change.notes}</TableCell>
                        <TableCell>
                          <Badge variant={statusVariant(change.status)}>{change.status}</Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {change.connection}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {(change.affectedTables ?? []).join(", ")}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {change.estAffected ?? 0} doc(s)
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="xs"
                            variant="outline"
                            render={
                              <Link to={PAGE_ROUTES.REQUEST_DETAIL(driver, change.id ?? "")} />
                            }
                          >
                            Review
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {!isPending && changes.length > 0 && total > PER_PAGE && (
              <Pagination page={page} perPage={PER_PAGE} total={total} onChange={setPage} />
            )}
          </div>
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
