import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Pagination } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { useApp } from "@/app/app-provider";
import { useDebounce } from "@/hooks/use-debounce";
import { CreateChangeForm } from "@/components/changes/create-change-form";
import { BatchChangeDialog } from "@/components/changes/batch-change-dialog";
import { ChangeTable } from "@/components/changes/change-table";
import { useChanges, type ChangeKind } from "@/hooks/use-data";

const PER_PAGE = 20;
const SKELETON_ROWS = [1, 2, 3, 4, 5, 6];

// ChangePage is the shared DML/DDL queue: raise inline, filter by instance,
// review the paginated single list.
export function ChangePage({ kind }: { kind: ChangeKind }) {
  const { can, registry } = useApp();
  const [filter, setFilter] = useState("");
  const query = useDebounce(filter.trim());
  const [page, setPage] = useState(1);
  const { changes, total, loading, submit, reload } = useChanges(kind, query || undefined, page);
  const label = kind.toUpperCase();
  // raising a change needs a write grant on some SQL instance; the form still
  // lets the server refuse a specific connection
  const mayRaise = registry.sql.connections.some((c) => can(c.name, "write"));

  // a changed filter starts back at the first page
  useEffect(() => setPage(1), [query]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-56">
          <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-7"
            placeholder="filter by instance"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        <span className="flex-1" />
        {mayRaise && <BatchChangeDialog onSubmitted={() => void reload()} />}
      </div>

      {mayRaise && <CreateChangeForm kind={kind} onSubmit={(input) => submit.run(input)} />}

      <Card>
        <CardContent className="flex flex-col gap-3 pt-4">
          <div className="flex flex-wrap items-baseline gap-2">
            <h2 className="text-sm font-medium">{label} changes</h2>
            <span className="text-xs text-muted-foreground">immediate and scheduled in one list</span>
            <span className="flex-1" />
            <span className="text-xs text-muted-foreground tabular-nums">{total}</span>
          </div>

          {loading ? (
            <div className="flex flex-col gap-2">
              {SKELETON_ROWS.map((n) => (
                <Skeleton key={n} className="h-9 w-full" />
              ))}
            </div>
          ) : total === 0 ? (
            <Empty
              title={`no ${kind} changes`}
              description={query ? "no change matches this instance filter" : "raise one and it will land here"}
            />
          ) : (
            <ChangeTable changes={changes} emptyHint={`no ${kind} changes on this page`} />
          )}

          {!loading && total > PER_PAGE && (
            <Pagination page={page} perPage={PER_PAGE} total={total} onChange={setPage} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}