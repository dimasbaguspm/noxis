import { Search } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Pagination } from "@/components/ui/pagination";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/app/auth-provider";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";
import { CreateChangeSheet } from "@/components/changes/create-change-sheet";
import { ChangeTable } from "@/components/changes/change-table";
import { Workspace } from "@/components/layout/workspace";
import { useChangeListQuery, useChangeListUrlParams, useSubmitChangeMutation, type ChangeKind } from "@/hooks/use-changes";

const PER_PAGE = 20;
const STATUS_OPTIONS = ["unverified", "validated", "approved", "applying", "applied", "failed", "rejected"];
const SKELETON_ROWS = [1, 2, 3, 4, 5, 6];

// ChangePage is the shared DML/DDL queue: filters and raising on the left, the paginated list on the right.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { canAccess } = useAuth();
  const { connections: sqlConnections } = useConnections(FAMILY.SQL);
  const {
    connection: filter,
    status,
    page,
    setConnection: setFilter,
    setStatus,
    setPage,
  } = useChangeListUrlParams();
  const { changes = [], total = 0, isPending, isFetching, refetch } = useChangeListQuery(kind);
  const submit = useSubmitChangeMutation();
  const label = kind.toUpperCase();
  // raising a change needs a write permission on some SQL instance; the form still lets the server refuse a specific connection
  const mayRaise = sqlConnections.some((c) => canAccess(c.name, "write"));

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        leftLabel={`${label} filters`}
        rightLabel={`${label} changes`}
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="change-filter">Filter by instance</Label>
              <InputGroup>
                <InputGroupAddon>
                  <Search />
                </InputGroupAddon>
                <InputGroupInput
                  id="change-filter"
                  placeholder="type to filter…"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                />
              </InputGroup>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(v) => setStatus(String(v))}>
                <SelectTrigger className="w-full">
                  <SelectValue />
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
            <Separator />
            {mayRaise && <CreateChangeSheet kind={kind} onSubmit={(input) => submit.mutateAsync(input)} />}
          </div>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto">
            {isPending ? (
              <div className="flex flex-col gap-2">
                {SKELETON_ROWS.map((n) => (
                  <Skeleton key={n} className="h-9 w-full" />
                ))}
              </div>
            ) : total === 0 ? (
              <Empty
                title={`no ${kind} changes`}
                description={filter.trim() ? "no change matches this instance filter" : "raise one and it will land here"}
              />
            ) : (
              <ChangeTable changes={changes} emptyHint={`no ${kind} changes on this page`} />
            )}

            {!isPending && total > PER_PAGE && (
              <Pagination page={page} perPage={PER_PAGE} total={total} onChange={setPage} />
            )}
          </div>
        }
      />
    </div>
  );
}
