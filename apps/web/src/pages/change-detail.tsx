import { Link, useParams } from "react-router-dom";
import { FileWarning, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { ChangeDetails, ChangeSqlReview } from "@/components/changes/change-review";
import { Workspace } from "@/components/layout/workspace";
import { useAuth } from "@/app/auth-provider";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import {
  useApproveChangeMutation,
  useApplyChangeMutation,
  useChangeGetQuery,
  useRejectChangeMutation,
  useScheduleChangeMutation,
  useSubmitChangeMutation,
  type ChangeKind,
} from "@/hooks/use-changes";
import { PAGE_ROUTES } from "@/lib/routes";
import { useCrumbLabel } from "@/app/crumb-provider";

// ChangeDetailPage owns loading and review permissions; the two visual halves live in components/changes/change-review.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "" } = useParams();
  const { canAccess, isInstanceAdmin } = useAuth();
  const { data: instancesData } = usePagedInstancesQuery();
  const detail = useChangeGetQuery(id);
  const change = detail.data;
  const gone = detail.isError;
  const fetching = detail.isPending;
  const { data: tables } = useConnectionSchemaQuery(change?.connection);
  const approve = useApproveChangeMutation();
  const apply = useApplyChangeMutation();
  const reject = useRejectChangeMutation();
  const submit = useSubmitChangeMutation();
  const schedule = useScheduleChangeMutation();
  // the trail names the change instead of its id; the envelope has no title, so notes or the touched tables speak for it
  useCrumbLabel(
    kind === "dml" ? PAGE_ROUTES.SQL.DML_DETAIL(id) : PAGE_ROUTES.SQL.DDL_DETAIL(id),
    change?.notes || change?.affectedTables?.join(", "),
  );

  if (fetching && !change) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Loader2 className="size-4 animate-spin" />
            Loading change {id.slice(0, 8)}
          </CardTitle>
        </CardHeader>
      </Card>
    );
  }

  if (gone || !change) return <Unavailable kind={kind} id={id} />;

  const instance = (instancesData?.instances ?? []).find((item) => item.name === change.connection);
  const mayReview = isInstanceAdmin(change.connection);
  const mayWrite = canAccess(change.connection, "write");
  const canApprove = mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = mayReview && change.status === "approved";
  const canReject = mayReview && change.status !== "applied" && change.status !== "rejected" && change.status !== "applying";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={<ChangeDetails change={change} instance={instance} />}
        right={
          <ChangeSqlReview
            change={change}
            tables={tables ?? []}
            kind={kind}
            mayWrite={mayWrite}
            canApprove={canApprove}
            canApply={canApply}
            canReject={canReject}
            onInverse={(input) => submit.mutateAsync(input)}
            onApprove={() => approve.mutate(id)}
            onApply={() => apply.mutate(id)}
            onReject={() => reject.mutate(id)}
            onSchedule={(at) => schedule.mutate({ id, scheduledAt: at })}
          />
        }
      />
    </div>
  );
}

// Unavailable is the state for an id that is not in the queue: the shared empty state, centered, with the way out.
function Unavailable({ kind, id }: { kind: ChangeKind; id: string }) {
  const queue = kind === "ddl" ? PAGE_ROUTES.SQL.DDL : PAGE_ROUTES.SQL.DML;
  return (
    <Empty
      icon={FileWarning}
      title={`Change ${id.slice(0, 8)} is not available`}
      description="The queue has no record with this id. It may have failed validation, belong to a removed connection, or come from another session."
      action={
        <div className="flex flex-wrap justify-center gap-2">
          <Button size="sm" render={<Link to={queue} />}>
            Back to {kind.toUpperCase()} queue
          </Button>
          <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.SQL.QUERY} />}>
            Go to Query
          </Button>
        </div>
      }
    />
  );
}