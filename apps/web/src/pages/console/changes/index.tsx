import { RefreshCw, Search } from "lucide-react";
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
import { useAuth } from "@/app/auth-provider";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { CreateChangeSheet } from "@/components/changes/create-change-sheet";
import { ChangeTable } from "@/components/changes/change-table";
import { InstanceFilter } from "@/components/filters/instance-filter";
import { QueuePanel } from "@/components/layout/queue-panel";
import { Workspace } from "@/components/layout/workspace";
import {
  useChangeListQuery,
  useChangeListUrlParams,
  useSubmitChangeMutation,
  type ChangeKind,
} from "@/hooks/use-changes";
import { cn } from "@/lib/utils";

const PER_PAGE = 20;
const STATUS_OPTIONS = [
  "unverified",
  "validated",
  "approved",
  "applying",
  "applied",
  "failed",
  "rejected",
];
const SKELETON_ROWS = [1, 2, 3, 4, 5, 6];

// ChangePage is the shared DML/DDL queue: filters on the left with the actions at their foot, the paginated list on the right.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { canAccess } = useAuth();
  const { connections: sqlConnections } = useDriverConnections();
  const { connections, name, status, page, setConnections, setName, setStatus, setPage } =
    useChangeListUrlParams();
  const { changes = [], total = 0, isPending, isFetching, refetch } = useChangeListQuery(kind);
  const submit = useSubmitChangeMutation();
  // raising a change needs a write permission on some SQL instance; the form still lets the server refuse a specific connection
  const mayRaise = sqlConnections.some((c) => canAccess(c.name, "write"));
  const narrowed = connections.length > 0 || name.trim() !== "";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <QueuePanel
            actions={
              <>
                {mayRaise && (
                  <CreateChangeSheet kind={kind} onSubmit={(input) => submit.mutateAsync(input)} />
                )}
                <Button variant="outline" onClick={() => void refetch()} disabled={isFetching}>
                  <RefreshCw className={cn(isFetching && "animate-spin")} /> Refresh
                </Button>
              </>
            }
          >
            <InstanceFilter
              options={sqlConnections.map((c) => c.name)}
              values={connections}
              onChange={setConnections}
            />
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="change-name">Name</Label>
              <Input
                id="change-name"
                placeholder="notes or statement"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(v) => setStatus(String(v))}>
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
          </QueuePanel>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto">
            {isPending ? (
              <div className="flex flex-col gap-2">
                {SKELETON_ROWS.map((n) => (
                  <Skeleton key={n} className="h-9 w-full" />
                ))}
              </div>
            ) : changes.length === 0 ? (
              <Empty
                className="flex-1"
                title={`no ${kind} changes`}
                description={
                  narrowed ? "No change matches these filters" : "Raise one and it will land here"
                }
              />
            ) : (
              <ChangeTable changes={changes} emptyHint={`no ${kind} changes on this page`} />
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
