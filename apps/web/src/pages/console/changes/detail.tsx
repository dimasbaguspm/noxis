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
import { PAGE_ROUTES } from "@/constants/routes";
import { useCrumbLabel } from "@/app/crumb-provider";

// ChangeDetailPage owns loading and review permissions; the two visual halves live in components/changes/change-review.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "", driver: routeDriver = "" } = useParams();
  const { canAccess, isInstanceAdmin } = useAuth();
  const { data: instancesData } = usePagedInstancesQuery();
  const detail = useChangeGetQuery(id);
  const change = detail.data;
  const gone = detail.isError;
  const fetching = detail.isPending;
  // the change names its driver; the route's driver stands in until it lands, so the schema read has a target
  const driver = change?.driver ?? routeDriver;
  const { data: tables } = useConnectionSchemaQuery(driver, change?.connection);
  const approve = useApproveChangeMutation();
  const apply = useApplyChangeMutation();
  const reject = useRejectChangeMutation();
  const submit = useSubmitChangeMutation();
  const schedule = useScheduleChangeMutation();
  // the trail names the change instead of its id; the envelope has no title, so notes or the touched tables speak for it
  useCrumbLabel(
    PAGE_ROUTES.CHANGE_DETAIL(driver, kind, id),
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

  if (gone || !change) return <Unavailable kind={kind} id={id} driver={routeDriver} />;

  const instance = (instancesData?.instances ?? []).find((item) => item.name === change.connection);
  const mayReview = isInstanceAdmin(change.connection);
  const mayWrite = canAccess(change.connection, "write");
  // a review action is addressed by the connection's driver, so it stays hidden until that driver is known
  const canApprove =
    !!driver && mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = !!driver && mayReview && change.status === "approved";
  const canReject =
    !!driver &&
    mayReview &&
    change.status !== "applied" &&
    change.status !== "rejected" &&
    change.status !== "applying";

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
            onApprove={() => approve.mutate({ id, driver: driver ?? "" })}
            onApply={() => apply.mutate({ id, driver: driver ?? "" })}
            onReject={() => reject.mutate({ id, driver: driver ?? "" })}
            onSchedule={(at) => schedule.mutate({ id, driver: driver ?? "", scheduledAt: at })}
          />
        }
      />
    </div>
  );
}

// Unavailable is the state for an id that is not in the queue: the shared empty state, centered, with the way out.
function Unavailable({ kind, id, driver }: { kind: ChangeKind; id: string; driver: string }) {
  const queue = PAGE_ROUTES.DRIVER(driver, kind);
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
          <Button
            size="sm"
            variant="outline"
            render={<Link to={PAGE_ROUTES.DRIVER(driver, "query")} />}
          >
            Go to Query
          </Button>
        </div>
      }
    />
  );
}
